package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	jwtpkg "github.com/sfedu-crm/pkg/jwt"
)

type mwUserRepo struct {
	getFn func(context.Context, int64) (*domain.User, error)
}

func (m *mwUserRepo) Create(context.Context, domain.CreateUserInput) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (m *mwUserRepo) GetByID(c context.Context, id int64) (*domain.User, error) {
	if m.getFn != nil {
		return m.getFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (m *mwUserRepo) GetByIDForUpdate(c context.Context, id int64) (*domain.User, error) {
	return m.GetByID(c, id)
}
func (m *mwUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (m *mwUserRepo) GetByPhone(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (m *mwUserRepo) List(context.Context, repository.UserFilter) ([]*domain.User, error) {
	return nil, nil
}
func (m *mwUserRepo) Update(context.Context, int64, domain.UpdateUserInput) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (m *mwUserRepo) UpdatePassword(context.Context, int64, string) error     { return nil }
func (m *mwUserRepo) AddBalance(context.Context, int64, int64) error          { return nil }
func (m *mwUserRepo) DebitBalance(context.Context, int64, int64) error        { return nil }
func (m *mwUserRepo) IncrementVisits(context.Context, int64, time.Time) error { return nil }
func (m *mwUserRepo) SetActive(context.Context, int64, bool) error            { return nil }

var _ repository.UserRepository = (*mwUserRepo)(nil)

func TestAuthMiddlewareSuccessUsesCurrentDatabaseRole(t *testing.T) {
	tm := jwtpkg.NewManager("12345678901234567890123456789012")
	token, err := tm.Generate(jwtpkg.Claims{UserID: 5, Role: "admin", TokenVersion: 2}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	repo := &mwUserRepo{getFn: func(context.Context, int64) (*domain.User, error) {
		return &domain.User{ID: 5, Role: domain.RoleManager, TokenVersion: 2, IsActive: true}, nil
	}}
	var gotID int64
	var gotRole domain.Role
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, _ = GetUserID(r.Context())
		gotRole, _ = GetRole(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "http://example.test/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	Auth(tm, repo)(next).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent || gotID != 5 || gotRole != domain.RoleManager {
		t.Fatalf("unexpected auth result code=%d id=%d role=%s", rr.Code, gotID, gotRole)
	}
}

func TestAuthMiddlewareRejectsMissingStaleInactiveAndBadToken(t *testing.T) {
	tm := jwtpkg.NewManager("12345678901234567890123456789012")
	valid, _ := tm.Generate(jwtpkg.Claims{UserID: 5, Role: "client", TokenVersion: 2}, time.Hour)
	cases := []struct {
		name, header string
		user         *domain.User
	}{
		{"missing", "", nil},
		{"bad", "Bearer garbage", nil},
		{"inactive", "Bearer " + valid, &domain.User{ID: 5, Role: domain.RoleClient, TokenVersion: 2, IsActive: false}},
		{"stale version", "Bearer " + valid, &domain.User{ID: 5, Role: domain.RoleClient, TokenVersion: 3, IsActive: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mwUserRepo{getFn: func(context.Context, int64) (*domain.User, error) {
				if tc.user == nil {
					return nil, domain.ErrNotFound
				}
				return tc.user, nil
			}}
			req := httptest.NewRequest(http.MethodGet, "http://example.test/x", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rr := httptest.NewRecorder()
			Auth(tm, repo)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { t.Fatal("next must not run") })).ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 got %d", rr.Code)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextRole, domain.RoleManager))
	rr := httptest.NewRecorder()
	RequireRole(domain.RoleManager, domain.RoleAdmin)(next).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("manager should pass, got %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	RequireRole(domain.RoleClient)(next).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("manager should be forbidden, got %d", rr.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(rr, req)
	expected := map[string]string{"X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY", "Referrer-Policy": "no-referrer"}
	for k, v := range expected {
		if rr.Header().Get(k) != v {
			t.Fatalf("%s=%q want %q", k, rr.Header().Get(k), v)
		}
	}
	csp := rr.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "frame-ancestors 'none'") || !strings.Contains(csp, "connect-src 'self'") {
		t.Fatalf("unexpected CSP: %s", csp)
	}
}

func TestIPRateLimitPerAddressAndReset(t *testing.T) {
	mw := IPRateLimit(2, 30*time.Millisecond)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	hit := func(addr string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = addr
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		return rr
	}
	if c := hit("10.0.0.1:123").Code; c != 204 {
		t.Fatal(c)
	}
	if c := hit("10.0.0.1:999").Code; c != 204 {
		t.Fatal(c)
	}
	rr := hit("10.0.0.1:321")
	if rr.Code != 429 || rr.Header().Get("Retry-After") == "" {
		t.Fatalf("expected limited with retry-after, got %d", rr.Code)
	}
	if c := hit("10.0.0.2:123").Code; c != 204 {
		t.Fatalf("different ip should pass: %d", c)
	}
	time.Sleep(40 * time.Millisecond)
	if c := hit("10.0.0.1:123").Code; c != 204 {
		t.Fatalf("window should reset: %d", c)
	}
}

func TestAIRateLimitPerUser(t *testing.T) {
	mw := AIRateLimit(1, time.Minute)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	hit := func(id int64) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req = req.WithContext(context.WithValue(req.Context(), ContextUserID, id))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		return rr
	}
	if c := hit(1).Code; c != 204 {
		t.Fatal(c)
	}
	if c := hit(1).Code; c != 429 {
		t.Fatalf("same user should be limited: %d", c)
	}
	if c := hit(2).Code; c != 204 {
		t.Fatalf("different user should pass: %d", c)
	}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatalf("missing actor should be 401, got %d", rr.Code)
	}
}

func TestMetricsCounts5xx(t *testing.T) {
	beforeReq := metricsRequests.Load()
	beforeErr := metricsErrors.Load()
	h := Metrics(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "x", http.StatusInternalServerError) }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if metricsRequests.Load() != beforeReq+1 || metricsErrors.Load() != beforeErr+1 {
		t.Fatalf("metrics not updated")
	}
	out := httptest.NewRecorder()
	MetricsHandler(out, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(out.Body.String(), "sfedu_http_requests_total") || !strings.Contains(out.Body.String(), "sfedu_http_5xx_total") {
		t.Fatalf("bad metrics body: %s", out.Body.String())
	}
}
