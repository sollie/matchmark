package main

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

func trustedProxyRealIP(trusted []netip.Prefix) func(http.Handler) http.Handler {
	contains := func(a netip.Addr) bool {
		for _, p := range trusted {
			if p.Contains(a) {
				return true
			}
		}
		return false
	}
	return func(next http.Handler) http.Handler {
		if len(trusted) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer, ok := peerAddr(r.RemoteAddr)
			if ok && contains(peer) {
				if ip, found := forwardedClient(r.Header.Get("X-Forwarded-For"), contains); found {
					r.RemoteAddr = net.JoinHostPort(ip.String(), "0")
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func forwardedClient(header string, trusted func(netip.Addr) bool) (netip.Addr, bool) {
	parts := strings.Split(header, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		addr, err := netip.ParseAddr(strings.TrimSpace(parts[i]))
		if err != nil {
			return netip.Addr{}, false
		}
		addr = addr.Unmap()
		if !trusted(addr) {
			return addr, true
		}
		if i == 0 {
			return addr, true
		}
	}
	return netip.Addr{}, false
}
