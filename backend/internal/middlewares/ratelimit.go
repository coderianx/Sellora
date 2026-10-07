package middlewares

import (
	"Sellora-Backend/internal/httpx"
	"Sellora-Backend/internal/store"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

// Fixed window. INCR and PEXPIRE stay in one script so two requests
// cannot both observe a missing key and skip the expiry.
var rateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("PTTL", KEYS[1])
if ttl < 0 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
	ttl = tonumber(ARGV[1])
end
return {current, ttl}
`)

func RateLimitByIP(
	limit int64,
	window time.Duration,
) func(http.Handler) http.Handler {
	if limit < 1 || window < time.Millisecond {
		return rejectInvalidRateLimit
	}

	windowMS := window.Milliseconds()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			ip, ok := realIP(r)

			if !ok {
				httpx.SendJSON(
					w, http.StatusBadRequest,
					map[string]any{
						"error": "cannot determine client IP",
					},
				)
				return
			}

			counts, err := rateLimitScript.Run(
				r.Context(),
				store.RedisClient,
				[]string{rateLimitKey(r, ip, limit, windowMS)},
				windowMS,
			).Int64Slice()

			if err != nil || len(counts) != 2 {
				httpx.SendJSON(
					w, http.StatusInternalServerError,
					map[string]any{
						"error": "Rate limit error",
					},
				)
				return
			}

			current := counts[0]
			ttlMS := counts[1]

			remaining := limit - current
			if remaining < 0 {
				remaining = 0
			}

			w.Header().Set(
				"X-RateLimit-Limit",
				fmt.Sprintf("%d", limit),
			)
			w.Header().Set(
				"X-RateLimit-Remaining",
				fmt.Sprintf("%d", remaining),
			)

			if current > limit {
				retryAfter := (ttlMS + 999) / 1000
				if retryAfter < 1 {
					retryAfter = 1
				}

				w.Header().Set(
					"Retry-After",
					fmt.Sprintf("%d", retryAfter),
				)

				httpx.SendJSON(
					w, http.StatusTooManyRequests,
					map[string]any{
						"error": "too many requests",
					},
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func rejectInvalidRateLimit(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		httpx.SendJSON(
			w, http.StatusInternalServerError,
			map[string]any{
				"error": "invalid rate limit",
			},
		)
	})
}

func rateLimitKey(
	r *http.Request,
	ip string,
	limit int64,
	windowMS int64,
) string {
	route := r.URL.Path

	if rc := chi.RouteContext(r.Context()); rc != nil {
		if pattern := rc.RoutePattern(); pattern != "" {
			route = pattern
		}
	}

	return fmt.Sprintf(
		"ratelimit:%s:%s:%d:%d:%s",
		r.Method,
		route,
		limit,
		windowMS,
		ip,
	)
}

// realIP is the TCP peer, with the port removed.
// X-Real-IP and X-Forwarded-For are accepted only from a loopback
// peer: that is a same-host reverse proxy. A remote client can set
// those headers itself, so trusting them would skip this limiter.
func realIP(r *http.Request) (string, bool) {
	peer, ok := parseIP(r.RemoteAddr)

	if !ok {
		return "", false
	}

	if peer.IsLoopback() {
		if ip, ok := proxyClientIP(r); ok {
			return ip.String(), true
		}
	}

	return peer.String(), true
}

func proxyClientIP(r *http.Request) (net.IP, bool) {
	if ip, ok := parseIP(r.Header.Get("X-Real-IP")); ok {
		return ip, true
	}

	if ip, ok := rightmostIP(r.Header.Get("X-Forwarded-For")); ok {
		return ip, true
	}

	if ip, ok := parseIP(r.Header.Get("CF-Connecting-IP")); ok {
		return ip, true
	}

	if ip, ok := parseIP(r.Header.Get("True-Client-IP")); ok {
		return ip, true
	}

	return nil, false
}

func rightmostIP(header string) (net.IP, bool) {
	parts := strings.Split(header, ",")

	for i := len(parts) - 1; i >= 0; i-- {
		if ip, ok := parseIP(parts[i]); ok {
			return ip, true
		}
	}

	return nil, false
}

func parseIP(value string) (net.IP, bool) {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"")

	if value == "" {
		return nil, false
	}

	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}

	value = strings.Trim(value, "[]")

	if i := strings.Index(value, "%"); i >= 0 {
		value = value[:i]
	}

	ip := net.ParseIP(value)
	if ip == nil {
		return nil, false
	}

	return ip, true
}
