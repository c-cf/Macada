package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// privateRanges are RFC 1918 + loopback + link-local CIDRs.
// Docker bridge networks always use private IPs within these ranges.
var privateRanges []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateRanges = append(privateRanges, block)
	}
}

// InternalOnly rejects requests whose source IP is not in a private network range.
// Apply this to /internal/* routes so that even if the port is exposed publicly,
// only containers on the Docker network (or localhost) can reach them.
func InternalOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		if ip == nil || !isPrivateIP(ip) {
			log.Warn().Str("remote", r.RemoteAddr).Msg("blocked non-internal request to internal endpoint")
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) net.IP {
	// Use RemoteAddr directly — do NOT trust X-Forwarded-For for internal routes.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might be just an IP without port
		host = strings.TrimSpace(r.RemoteAddr)
	}
	return net.ParseIP(host)
}

func isPrivateIP(ip net.IP) bool {
	for _, block := range privateRanges {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
