package service

import (
	"context"
	"errors"
	"testing"
)

func TestGuestContactValidation(t *testing.T) {
	good := GuestContact{Name: " Jane ", Phone: "+63 912 345 6789", Email: "jane@example.test", Notes: "Outdoor ceremony"}
	result, e := cleanGuestContact(good)
	if e != nil || result.Name != "Jane" {
		t.Fatal(result, e)
	}
	for _, bad := range []GuestContact{{Name: "", Phone: "12345"}, {Name: "Jane", Phone: "abcde"}, {Name: "Jane", Phone: "12345", Email: "invalid"}, {Name: "Jane\nDoe", Phone: "12345"}} {
		if _, e := cleanGuestContact(bad); !errors.Is(e, ErrInvalidBooking) {
			t.Fatalf("accepted invalid contact: %+v", bad)
		}
	}
}
func TestGuestCapabilityRequiresRandomToken(t *testing.T) {
	s := NewBusinessService(nil)
	for _, token := range []string{"", "123", "customer@example.com"} {
		if _, e := s.GuestVisitor(context.Background(), token, true); !errors.Is(e, ErrInvalidSession) {
			t.Fatal(e)
		}
	}
}
