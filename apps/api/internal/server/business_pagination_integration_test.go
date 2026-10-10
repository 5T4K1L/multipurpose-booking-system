package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestBusinessPaginationIntegration(t *testing.T) {
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
	users := []int64{}
	businesses := []pgtype.UUID{}
	defer func() {
		c, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, id := range businesses {
			if _, e := pool.Exec(c, "DELETE FROM businesses WHERE id=$1", id); e != nil {
				t.Error(e)
			}
		}
		if _, e := pool.Exec(c, "DELETE FROM user_sessions WHERE user_id=ANY($1::bigint[])", users); e != nil {
			t.Error(e)
		}
		if _, e := pool.Exec(c, "DELETE FROM users WHERE id=ANY($1::bigint[])", users); e != nil {
			t.Error(e)
		}
	}()
	rows, err := pool.Query(ctx, "INSERT INTO users(email,username) SELECT $1 || n || '@example.invalid','pagination fixture' FROM generate_series(1,105) n RETURNING id", fmt.Sprintf("page-%d-", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id int64
		if e := rows.Scan(&id); e != nil {
			t.Fatal(e)
		}
		users = append(users, id)
	}
	rows.Close()
	if rows.Err() != nil {
		t.Fatal(rows.Err())
	}
	rows, err = pool.Query(ctx, "INSERT INTO businesses(name,timezone) SELECT 'pagination fixture','UTC' FROM generate_series(1,105) RETURNING id")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id pgtype.UUID
		if e := rows.Scan(&id); e != nil {
			t.Fatal(e)
		}
		businesses = append(businesses, id)
	}
	rows.Close()
	if rows.Err() != nil {
		t.Fatal(rows.Err())
	}
	if _, err = pool.Exec(ctx, "INSERT INTO business_memberships(business_id,user_id,role) SELECT unnest($1::uuid[]),$2,'BUSINESS_OWNER'", businesses, users[0]); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "INSERT INTO business_memberships(business_id,user_id,role) SELECT $1,unnest($2::bigint[]),'STAFF'", businesses[0], users[1:]); err != nil {
		t.Fatal(err)
	}
	sessions, err := service.NewSessionService(repository.NewSessionTransactions(pool), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := sessions.Create(ctx, users[0])
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewBusinessService(repository.NewBusinessTransactions(pool))
	h, err := NewBusinessHandler(sessions, svc, "http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	request := func(path string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("GET", "/api/v1/businesses"+path, nil).WithContext(ctx)
		r.AddCookie(&http.Cookie{Name: "booking_session", Value: issued.Token})
		r.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("got %d want %d: %s", w.Code, want, w.Body.String())
		}
		return w
	}
	for _, query := range []string{"?limit=0", "?limit=101", "?limit=-1", "?limit=1&limit=2", "?cursor=", "?cursor=bad", "?cursor=a&cursor=b"} {
		request(query, 400)
	}
	first := request("", 200)
	var firstItems []service.BusinessView
	if err := json.Unmarshal(first.Body.Bytes(), &firstItems); err != nil {
		t.Fatal(err)
	}
	if len(firstItems) != 100 || first.Header().Get("X-Next-Cursor") == "" || first.Header().Get("Access-Control-Expose-Headers") != "X-Next-Cursor" {
		t.Fatal("default page/header incorrect")
	}
	seen := map[pgtype.UUID]bool{}
	cursor := ""
	for page := 0; page < 10; page++ {
		path := "?limit=17"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		w := request(path, 200)
		var items []service.BusinessView
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		if len(items) > 17 {
			t.Fatal("page too large")
		}
		for _, item := range items {
			if seen[item.ID] {
				t.Fatal("duplicate business across equal-timestamp pages")
			}
			seen[item.ID] = true
		}
		cursor = w.Header().Get("X-Next-Cursor")
		if cursor == "" {
			break
		}
	}
	if len(seen) != 105 || cursor != "" {
		t.Fatal("businesses were truncated", len(seen))
	}
	idJSON, _ := businesses[0].MarshalJSON()
	path := "/" + string(idJSON[1:len(idJSON)-1]) + "/memberships"
	seenUsers := map[int64]bool{}
	cursor = ""
	var memberCursor string
	for page := 0; page < 10; page++ {
		query := path + "?limit=19"
		if cursor != "" {
			query += "&cursor=" + cursor
		}
		w := request(query, 200)
		var items []service.MembershipView
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		for _, item := range items {
			if seenUsers[item.UserID] {
				t.Fatal("duplicate member")
			}
			seenUsers[item.UserID] = true
		}
		cursor = w.Header().Get("X-Next-Cursor")
		if page == 0 {
			memberCursor = cursor
		}
		if cursor == "" {
			break
		}
	}
	if len(seenUsers) != 105 || cursor != "" {
		t.Fatal("memberships were truncated", len(seenUsers))
	}
	request(path+"?cursor="+first.Header().Get("X-Next-Cursor"), 400)
	otherJSON, _ := businesses[1].MarshalJSON()
	request("/"+string(otherJSON[1:len(otherJSON)-1])+"/memberships?cursor="+memberCursor, 400)
	// A valid cursor never preserves a revoked permission.
	if err := svc.ChangeMember(ctx, users[0], businesses[0], users[1], service.RoleBusinessAdmin, false); err != nil {
		t.Fatal(err)
	}
	p, err := svc.MembersPage(ctx, users[1], businesses[0], service.PageRequest{Limit: 1})
	if err != nil || p.NextCursor == "" {
		t.Fatal(err)
	}
	if err := svc.ChangeMember(ctx, users[0], businesses[0], users[1], "", true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MembersPage(ctx, users[1], businesses[0], service.PageRequest{Limit: 1, Cursor: p.NextCursor}); err != service.ErrForbidden {
		t.Fatal("cursor retained revoked access", err)
	}
	// Deleting the boundary row does not invalidate keyset continuation.
	if _, err := pool.Exec(ctx, "DELETE FROM businesses WHERE id=$1", firstItems[len(firstItems)-1].ID); err != nil {
		t.Fatal(err)
	}
	w := request("?cursor="+first.Header().Get("X-Next-Cursor"), 200)
	var tail []service.BusinessView
	if err := json.Unmarshal(w.Body.Bytes(), &tail); err != nil || len(tail) != 5 {
		t.Fatal("deleted boundary broke continuation", len(tail), err)
	}
}
