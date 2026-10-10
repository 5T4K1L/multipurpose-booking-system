package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestBusinessIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ids := []int64{}
	businessIDs := []string{}
	defer func() {
		c, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, id := range businessIDs {
			if _, e := pool.Exec(c, "DELETE FROM businesses WHERE id=$1", id); e != nil {
				t.Error(e)
			}
		}
		for _, id := range ids {
			for _, table := range []string{"user_sessions", "user_platform_roles", "users"} {
				key := "user_id"
				if table == "users" {
					key = "id"
				}
				if _, e := pool.Exec(c, "DELETE FROM "+table+" WHERE "+key+"=$1", id); e != nil {
					t.Error(e)
				}
			}
		}
	}()
	sessions, err := service.NewSessionService(repository.NewSessionTransactions(pool), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tokens := []string{}
	for i := 0; i < 3; i++ {
		var id int64
		if err := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'tenant fixture') RETURNING id", fmt.Sprintf("tenant-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
		issued, e := sessions.Create(ctx, id)
		if e != nil {
			t.Fatal(e)
		}
		tokens = append(tokens, issued.Token)
	}
	rollbackName := fmt.Sprintf("rollback-%d", time.Now().UnixNano())
	rollbackErr := errors.New("intentional fixture rollback")
	err = repository.NewBusinessTransactions(pool).WithinBusiness(ctx, func(q service.BusinessStore) error {
		created, e := q.CreateBusiness(ctx, db.CreateBusinessParams{Name: rollbackName, Timezone: "UTC"})
		if e != nil {
			return e
		}
		if e = q.PutBusinessMembership(ctx, db.PutBusinessMembershipParams{BusinessID: created.ID, UserID: ids[0], Role: "BUSINESS_OWNER"}); e != nil {
			return e
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM businesses WHERE name=$1", rollbackName).Scan(&count); err != nil || count != 0 {
		t.Fatal("business transaction did not roll back", count, err)
	}
	svc := service.NewBusinessService(repository.NewBusinessTransactions(pool))
	h, err := NewBusinessHandler(sessions, svc, "http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	request := func(actor int, method, path, body, origin string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/api/v1/businesses"+path, strings.NewReader(body)).WithContext(ctx)
		if actor >= 0 {
			r.AddCookie(&http.Cookie{Name: "booking_session", Value: tokens[actor]})
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-User-ID", fmt.Sprint(ids[0]))
		r.Header.Set("X-Role", "BUSINESS_OWNER")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s actor %d got %d want %d: %s", method, path, actor, w.Code, want, w.Body.String())
		}
		return w
	}
	const origin = "http://localhost:3000"
	request(-1, "GET", "", "", "", 401)
	request(0, "POST", "", `{"name":"A","timezone":"UTC"}`, "", 403)
	request(0, "POST", "", `{"name":"A","timezone":"UTC","owner_id":1}`, origin, 400)
	request(0, "POST", "", `{"name":"A","timezone":"not/a/zone"}`, origin, 400)
	for i := 0; i < 2; i++ {
		w := request(i, "POST", "", `{"name":"Fixture","timezone":"Asia/Shanghai"}`, origin, 201)
		var b struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
			t.Fatal(err)
		}
		if b.Role != "BUSINESS_OWNER" {
			t.Fatal(b)
		}
		businessIDs = append(businessIDs, b.ID)
	}
	a, b := "/"+businessIDs[0], "/"+businessIDs[1]
	request(0, "GET", a, "", "", 200)
	request(0, "GET", b, "", "", 403)
	request(1, "GET", a, "", "", 403)
	w := request(0, "GET", "", "", "", 200)
	if strings.Contains(w.Body.String(), businessIDs[1]) {
		t.Fatal("cross-tenant listing")
	}
	if _, err := pool.Exec(ctx, "INSERT INTO user_platform_roles(user_id,role) VALUES($1,'PLATFORM_ADMIN')", ids[2]); err != nil {
		t.Fatal(err)
	}
	request(2, "GET", a, "", "", 403)
	member := a + "/memberships/" + fmt.Sprint(ids[2])
	request(0, "PUT", member, `{"role":"PLATFORM_ADMIN"}`, origin, 400)
	request(0, "PUT", member, `{"role":"BUSINESS_OWNER"}`, origin, 400)
	request(1, "PUT", member, `{"role":"STAFF"}`, origin, 403)
	request(0, "PUT", member, `{"role":"STAFF"}`, origin, 204)
	request(2, "GET", a, "", "", 200)
	request(2, "GET", a+"/memberships", "", "", 403)
	request(2, "PUT", member, `{"role":"BUSINESS_ADMIN"}`, origin, 403)
	request(0, "PUT", member, `{"role":"BUSINESS_ADMIN"}`, origin, 204)
	request(2, "GET", a+"/memberships", "", "", 200)
	owner := a + "/memberships/" + fmt.Sprint(ids[0])
	request(0, "DELETE", owner, "", origin, 403)
	request(0, "PUT", owner, `{"role":"STAFF"}`, origin, 403)
	request(0, "DELETE", member, "", origin, 204)
	request(2, "GET", a, "", "", 403)
	if _, err := pool.Exec(ctx, "UPDATE businesses SET status='disabled' WHERE id=$1", businessIDs[0]); err != nil {
		t.Fatal(err)
	}
	request(0, "GET", a, "", "", 403)
	if _, err := pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", ids[1]); err != nil {
		t.Fatal(err)
	}
	request(1, "GET", b, "", "", 401)
}
