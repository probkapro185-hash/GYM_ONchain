package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/pkg/jwt"
)

type contextKey string

const (
	ContextUserID    contextKey = "user_id"
	ContextRole      contextKey = "role"
	ContextRequestID contextKey = "request_id"
)

func Auth(tokenMgr *jwt.Manager, users repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if tokenString == "" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			claims, err := tokenMgr.Parse(tokenString)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			// Re-read the account on every authenticated request. This immediately
			// enforces deactivation and current DB role instead of trusting stale JWT claims.
			user, err := users.GetByID(r.Context(), claims.UserID)
			if err != nil || !user.IsActive || user.TokenVersion != claims.TokenVersion {
				writeJSONError(w, http.StatusUnauthorized, "account unavailable")
				return
			}

			ctx := context.WithValue(r.Context(), ContextUserID, user.ID)
			ctx = context.WithValue(ctx, ContextRole, user.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	allowed := make(map[domain.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := GetRole(r.Context())
			if !ok {
				writeJSONError(w, http.StatusForbidden, "forbidden")
				return
			}
			if _, ok := allowed[role]; !ok {
				writeJSONError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	allowAny := false
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "*" {
			allowAny = true
		} else if origin != "" {
			allowed[origin] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			sameOrigin := false
			if origin != "" {
				if parsed, err := url.Parse(origin); err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host == r.Host {
					sameOrigin = true
				}
			}
			w.Header().Add("Vary", "Origin")
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")

			if origin != "" {
				if sameOrigin {
					// CORS is a cross-origin policy. Never block requests from the web UI
					// served by this same host just because it is absent from the external allowlist.
				} else if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				} else if allowAny {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					writeJSONError(w, http.StatusForbidden, "origin not allowed")
					return
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: blob: https://images.unsplash.com; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			requestID, _ := GetRequestID(r.Context())
			log.Info("request", "request_id", requestID, "method", r.Method, "path", r.URL.Path,
				"status", rw.status, "duration", time.Since(start).String())
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.wroteHeader = true
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(p)
}

func GetUserID(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(ContextUserID).(int64)
	return v, ok
}

func GetRole(ctx context.Context) (domain.Role, bool) {
	v, ok := ctx.Value(ContextRole).(domain.Role)
	return v, ok
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" || len(requestID) > 80 {
			var b [12]byte
			if _, err := rand.Read(b[:]); err == nil {
				requestID = hex.EncodeToString(b[:])
			} else {
				requestID = strconv.FormatInt(time.Now().UnixNano(), 36)
			}
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), ContextRequestID, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ContextRequestID).(string)
	return v, ok
}

var metricsRequests atomic.Uint64
var metricsErrors atomic.Uint64
var metricsDurationNs atomic.Uint64

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		metricsRequests.Add(1)
		metricsDurationNs.Add(uint64(time.Since(start).Nanoseconds()))
		if rw.status >= 500 {
			metricsErrors.Add(1)
		}
	})
}

func MetricsHandler(w http.ResponseWriter, _ *http.Request) {
	requests := metricsRequests.Load()
	errorsCount := metricsErrors.Load()
	duration := metricsDurationNs.Load()
	avg := float64(0)
	if requests > 0 {
		avg = float64(duration) / float64(requests) / 1e9
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = fmt.Fprintf(w, "# HELP sfedu_http_requests_total Total HTTP requests.\n# TYPE sfedu_http_requests_total counter\nsfedu_http_requests_total %d\n", requests)
	_, _ = fmt.Fprintf(w, "# HELP sfedu_http_5xx_total Total HTTP 5xx responses.\n# TYPE sfedu_http_5xx_total counter\nsfedu_http_5xx_total %d\n", errorsCount)
	_, _ = fmt.Fprintf(w, "# HELP sfedu_http_request_duration_seconds_avg Average request duration.\n# TYPE sfedu_http_request_duration_seconds_avg gauge\nsfedu_http_request_duration_seconds_avg %.6f\n", avg)
}

func AuditChanges(crm repository.CRMRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || rw.status >= 400 {
				return
			}
			userID, okID := GetUserID(r.Context())
			role, okRole := GetRole(r.Context())
			requestID, _ := GetRequestID(r.Context())
			var actorID *int64
			if okID {
				actorID = &userID
			}
			actorRole := ""
			if okRole {
				actorRole = string(role)
			}
			if err := crm.RecordAudit(context.Background(), actorID, actorRole, requestID, r.Method, r.URL.Path, rw.status); err != nil {
				slog.Error("audit write failed", "error", err, "request_id", requestID)
			}
		})
	}
}

func AIRateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	if limit <= 0 {
		limit = 20
	}
	if window <= 0 {
		window = time.Minute
	}
	type bucket struct {
		count int
		reset time.Time
	}
	var mu sync.Mutex
	buckets := make(map[int64]bucket)
	lastCleanup := time.Now()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserID(r.Context())
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			now := time.Now()
			mu.Lock()
			if now.Sub(lastCleanup) >= 10*time.Minute {
				for id, item := range buckets {
					if now.After(item.reset.Add(window)) {
						delete(buckets, id)
					}
				}
				lastCleanup = now
			}
			item := buckets[userID]
			if item.reset.IsZero() || !now.Before(item.reset) {
				item = bucket{count: 0, reset: now.Add(window)}
			}
			if item.count >= limit {
				retryAfter := int(time.Until(item.reset).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				mu.Unlock()
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				writeJSONError(w, http.StatusTooManyRequests, "AI rate limit exceeded")
				return
			}
			item.count++
			buckets[userID] = item
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

// IPRateLimit is a small in-process limiter for unauthenticated endpoints such as
// login/application submission. It deliberately uses RemoteAddr and does not trust
// spoofable forwarding headers; deployments behind a proxy should additionally
// enforce rate limits at the trusted ingress layer.
func IPRateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	type bucket struct {
		count int
		reset time.Time
	}
	var mu sync.Mutex
	buckets := make(map[string]bucket)
	lastCleanup := time.Now()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" {
				key = host
			}
			if key == "" {
				key = "unknown"
			}
			now := time.Now()
			mu.Lock()
			if now.Sub(lastCleanup) >= 10*time.Minute {
				for id, item := range buckets {
					if now.After(item.reset.Add(window)) {
						delete(buckets, id)
					}
				}
				lastCleanup = now
			}
			item := buckets[key]
			if item.reset.IsZero() || !now.Before(item.reset) {
				item = bucket{reset: now.Add(window)}
			}
			if item.count >= limit {
				retryAfter := int(time.Until(item.reset).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				mu.Unlock()
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			item.count++
			buckets[key] = item
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}
