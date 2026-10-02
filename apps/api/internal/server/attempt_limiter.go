package server

import (
	"net"
	"net/netip"
	"sync"
	"time"
)

type registrationWindow struct {
	count int
	until time.Time
}
type attemptLimiter struct {
	mu      sync.Mutex
	windows map[string]registrationWindow
	now     func() time.Time
	limit   int
}

func newAttemptLimiter(limit int) attemptLimiter {
	return attemptLimiter{windows: make(map[string]registrationWindow), now: time.Now, limit: limit}
}

// Only the transport peer is trusted; forwarding headers cannot bypass limits.
func (h *attemptLimiter) allow(peer string) bool {
	host, _, err := net.SplitHostPort(peer)
	if err != nil {
		host = peer
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		host = "unknown"
	} else {
		host = address.Unmap().String()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	window, exists := h.windows[host]
	if !exists && len(h.windows) >= 10000 {
		for key, value := range h.windows {
			if !now.Before(value.until) {
				delete(h.windows, key)
			}
		}
		if len(h.windows) >= 10000 {
			return false
		}
	}
	if !now.Before(window.until) {
		window = registrationWindow{until: now.Add(15 * time.Minute)}
	}
	if window.count >= h.limit {
		return false
	}
	window.count++
	h.windows[host] = window
	return true
}
