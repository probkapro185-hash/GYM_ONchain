package httphandler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sfedu-crm/internal/domain"
)

func TestDecodeStrictJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	cases := []struct {
		name, body string
		ok         bool
	}{
		{"valid", `{"name":"x"}`, true},
		{"unknown field", `{"name":"x","extra":1}`, false},
		{"two objects", `{"name":"x"}{"name":"y"}`, false},
		{"malformed", `{"name":`, false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			var out payload
			err := decode(rr, req, &out)
			if tc.ok && err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDecodeRejectsOversizedBody(t *testing.T) {
	body := bytes.Repeat([]byte("a"), maxRequestBody+100)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	var out map[string]any
	if err := decode(rr, req, &out); err == nil {
		t.Fatal("expected oversized body error")
	}
}

func TestHandleErrorStatusMapping(t *testing.T) {
	cases := []struct {
		err      error
		status   int
		contains string
	}{
		{domain.ErrNotFound, 404, "not found"},
		{domain.ErrNoActiveSubscription, 404, "no active subscription"},
		{domain.ErrAlreadyExists, 409, "already exists"},
		{domain.ErrConflict, 409, "conflict"},
		{domain.ErrInvalidTransition, 409, "invalid status transition"},
		{domain.ErrInvalidInput, 400, "invalid input"},
		{domain.ErrInvalidPhone, 400, "invalid phone"},
		{domain.ErrInvalidEmail, 400, "invalid email"},
		{domain.ErrUnauthorized, 401, "unauthorized"},
		{domain.ErrInvalidPassword, 401, "unauthorized"},
		{domain.ErrForbidden, 403, "forbidden"},
		{domain.ErrInsufficientFunds, 402, "insufficient funds"},
		{domain.ErrServiceUnavailable, 503, "AI service unavailable"},
		{errors.New("secret db message"), 500, "internal server error"},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		handleError(rr, tc.err)
		if rr.Code != tc.status || !strings.Contains(rr.Body.String(), tc.contains) {
			t.Fatalf("err=%v got status=%d body=%q", tc.err, rr.Code, rr.Body.String())
		}
		if tc.status == 500 && strings.Contains(rr.Body.String(), "secret db message") {
			t.Fatal("internal error leaked")
		}
	}
}

func TestParseIDFromPathAndPositiveInt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	req.SetPathValue("id", "123")
	if id, err := parseIDFromPath(req); err != nil || id != 123 {
		t.Fatalf("got %d %v", id, err)
	}
	for _, v := range []string{"", "0", "-1", "abc"} {
		req.SetPathValue("id", v)
		if _, err := parseIDFromPath(req); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("%q expected invalid: %v", v, err)
		}
	}
	if id, err := parsePositiveInt64("9"); err != nil || id != 9 {
		t.Fatalf("positive parse: %d %v", id, err)
	}
	for _, v := range []string{"0", "-2", "x"} {
		if _, err := parsePositiveInt64(v); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("%q expected invalid", v)
		}
	}
}

func TestParseDateUTC(t *testing.T) {
	d, err := parseDate("2026-09-15")
	if err != nil {
		t.Fatal(err)
	}
	if d.Location() != time.UTC || d.Year() != 2026 || d.Month() != 9 || d.Day() != 15 {
		t.Fatalf("unexpected date %v", d)
	}
	if _, err := parseDate("15.09.2026"); err == nil {
		t.Fatal("wrong format should fail")
	}
}
