package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/service"
	"os"
	"testing"
	"time"
)

func TestTenantConcurrentRevocation(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var owner, member int64
	ids := []int64{}
	businesses := []pgtype.UUID{}
	defer func() {
		c, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, id := range businesses {
			if _, e := pool.Exec(c, "DELETE FROM businesses WHERE id=$1", id); e != nil {
				t.Error(e)
			}
		}
		for _, id := range ids {
			if _, e := pool.Exec(c, "DELETE FROM users WHERE id=$1", id); e != nil {
				t.Error(e)
			}
		}
	}()
	for i, dest := range []*int64{&owner, &member} {
		if err := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'concurrency fixture') RETURNING id", fmt.Sprintf("tenant-lock-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(dest); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, *dest)
	}
	svc := service.NewBusinessService(NewBusinessTransactions(pool))
	for i := 0; i < 2; i++ {
		b, e := svc.Create(ctx, owner, service.BusinessInput{Name: "concurrency fixture", Timezone: "UTC"})
		if e != nil {
			t.Fatal(e)
		}
		businesses = append(businesses, b.ID)
	}
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("tenant-waiter-%d", time.Now().UnixNano())
	config.ConnConfig.RuntimeParams["application_name"] = name
	waiterPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer waiterPool.Close()
	waiter := service.NewBusinessService(NewBusinessTransactions(waiterPool))
	for _, mode := range []string{"revoke", "demote"} {
		t.Run(mode, func(t *testing.T) {
			if err := svc.ChangeMember(ctx, owner, businesses[0], member, service.RoleBusinessAdmin, false); err != nil {
				t.Fatal(err)
			}
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(context.Background())
			if _, err = tx.Exec(ctx, "SELECT id FROM businesses WHERE id=$1 FOR UPDATE", businesses[0]); err != nil {
				t.Fatal(err)
			}
			if mode == "revoke" {
				_, err = tx.Exec(ctx, "DELETE FROM business_memberships WHERE business_id=$1 AND user_id=$2", businesses[0], member)
			} else {
				_, err = tx.Exec(ctx, "UPDATE business_memberships SET role='STAFF' WHERE business_id=$1 AND user_id=$2", businesses[0], member)
			}
			if err != nil {
				t.Fatal(err)
			}
			result := make(chan error, 1)
			go func() { _, e := waiter.MembersPage(ctx, member, businesses[0], service.PageRequest{}); result <- e }()
			deadline := time.Now().Add(5 * time.Second)
			for {
				var waiting bool
				if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1 AND wait_event_type='Lock')", name).Scan(&waiting); err != nil {
					t.Fatal(err)
				}
				if waiting {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("request never waited on business lock")
				}
				select {
				case e := <-result:
					t.Fatalf("request passed pending authorization change: %v", e)
				case <-time.After(10 * time.Millisecond):
				}
			}
			// A lock on one tenant must not block an unrelated tenant.
			otherCtx, stop := context.WithTimeout(ctx, time.Second)
			_, err = svc.Get(otherCtx, owner, businesses[1])
			stop()
			if err != nil {
				t.Fatal("unrelated tenant blocked", err)
			}
			// Cancellation of another waiter must not leak a callback or transaction.
			short, stop := context.WithTimeout(ctx, 100*time.Millisecond)
			called := false
			err = waiter.WithinTenant(short, owner, businesses[0], []service.Role{service.RoleBusinessOwner}, func(service.BusinessStore, service.TenantAccess) error { called = true; return nil })
			stop()
			if err == nil || called {
				t.Fatal("cancelled wait ran callback", err)
			}
			if err = tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case e := <-result:
				if !errors.Is(e, service.ErrForbidden) {
					t.Fatal("stale membership authorized", e)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if _, e := svc.Get(ctx, owner, businesses[0]); e != nil {
				t.Fatal("lock leaked", e)
			}
		})
	}
}
