package server

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

func sessionCookieSecurity(origin string) (bool, error) {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" || origin != parsed.Scheme+"://"+parsed.Host {
		return false, errors.New("WEB_ORIGIN must be one exact HTTP(S) origin without a path")
	}
	secure := parsed.Scheme == "https"
	ip := net.ParseIP(parsed.Hostname())
	loopback := strings.EqualFold(parsed.Hostname(), "localhost") || (ip != nil && ip.IsLoopback())
	if !secure && (parsed.Scheme != "http" || !loopback) {
		return false, errors.New("WEB_ORIGIN must use HTTPS outside loopback development")
	}
	return secure, nil
}

func sessionCookieName(secure bool) string {
	if secure {
		return "__Host-booking_session"
	}
	return "booking_session"
}
