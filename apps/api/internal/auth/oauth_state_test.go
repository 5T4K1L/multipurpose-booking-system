package auth

import (
	"testing"
	"time"
)

func TestGenerateOAuthState(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

	first, err := GenerateOAuthState(now)
	if err != nil {
		t.Fatalf("generate first OAuth state: %v", err)
	}

	second, err := GenerateOAuthState(now)
	if err != nil {
		t.Fatalf("generate second OAuth state: %v", err)
	}

	if first.State == "" || first.Nonce == "" {
		t.Fatal("expected state and nonce to be generated")
	}

	if first.State == second.State {
		t.Fatal("expected unique state values")
	}

	if first.Nonce == second.Nonce {
		t.Fatal("expected unique nonce values")
	}

	if first.CreatedAt != now {
		t.Fatalf("expected CreatedAt %v, got %v", now, first.CreatedAt)
	}
}

func TestValidateOAuthState(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		received string
		want     bool
	}{
		{
			name:     "matching state",
			expected: "expected-state",
			received: "expected-state",
			want:     true,
		},
		{
			name:     "wrong state",
			expected: "expected-state",
			received: "wrong-state",
			want:     false,
		},
		{
			name:     "empty expected state",
			expected: "",
			received: "state",
			want:     false,
		},
		{
			name:     "empty received state",
			expected: "state",
			received: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateOAuthState(tt.expected, tt.received); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestValidateOAuthStateAge(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	maxAge := 10 * time.Minute

	tests := []struct {
		name      string
		createdAt time.Time
		now       time.Time
		maxAge    time.Duration
		want      bool
	}{
		{
			name:      "fresh state",
			createdAt: now.Add(-5 * time.Minute),
			now:       now,
			maxAge:    maxAge,
			want:      true,
		},
		{
			name:      "expired state",
			createdAt: now.Add(-11 * time.Minute),
			now:       now,
			maxAge:    maxAge,
			want:      false,
		},
		{
			name:      "future state",
			createdAt: now.Add(time.Minute),
			now:       now,
			maxAge:    maxAge,
			want:      false,
		},
		{
			name:      "zero max age",
			createdAt: now,
			now:       now,
			maxAge:    0,
			want:      false,
		},
		{
			name:      "zero creation time",
			createdAt: time.Time{},
			now:       now,
			maxAge:    maxAge,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateOAuthStateAge(tt.createdAt, tt.now, tt.maxAge); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
