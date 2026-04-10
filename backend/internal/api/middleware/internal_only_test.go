package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		// Docker bridge typical
		{"172.18.0.3", true},
		{"172.17.0.2", true},

		// RFC 1918
		{"10.0.0.1", true},
		{"192.168.1.100", true},

		// Loopback
		{"127.0.0.1", true},
		{"::1", true},

		// Public IPs — should be rejected
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"203.0.113.50", false},

		// Edge of private range
		{"172.15.255.255", false},
		{"172.16.0.0", true},
		{"172.31.255.255", true},
		{"172.32.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tt.ip)
			}
			got := isPrivateIP(ip)
			if got != tt.want {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestInternalOnly_AllowsPrivateIP(t *testing.T) {
	handler := InternalOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/internal/v1/sandbox/sesn_01/llm", nil)
	req.RemoteAddr = "172.18.0.3:54321"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for private IP, got %d", rr.Code)
	}
}

func TestInternalOnly_BlocksPublicIP(t *testing.T) {
	handler := InternalOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/internal/v1/sandbox/sesn_01/llm", nil)
	req.RemoteAddr = "203.0.113.50:54321"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for public IP, got %d", rr.Code)
	}
}

func TestInternalOnly_AllowsLocalhost(t *testing.T) {
	handler := InternalOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/internal/v1/sandbox/sesn_01/llm", nil)
	req.RemoteAddr = "127.0.0.1:54321"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for localhost, got %d", rr.Code)
	}
}
