package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sfedu-crm/internal/domain"
)

func TestMapDBError(t *testing.T) {
	sentinel := errors.New("boom")
	cases := []struct {
		name   string
		in     error
		target error
	}{
		{"nil", nil, nil},
		{"not found", pgx.ErrNoRows, domain.ErrNotFound},
		{"unique", &pgconn.PgError{Code: "23505"}, domain.ErrAlreadyExists},
		{"not null", &pgconn.PgError{Code: "23502"}, domain.ErrInvalidInput},
		{"foreign key", &pgconn.PgError{Code: "23503"}, domain.ErrInvalidInput},
		{"check", &pgconn.PgError{Code: "23514"}, domain.ErrInvalidInput},
		{"too long", &pgconn.PgError{Code: "22001"}, domain.ErrInvalidInput},
		{"numeric range", &pgconn.PgError{Code: "22003"}, domain.ErrInvalidInput},
		{"invalid text", &pgconn.PgError{Code: "22P02"}, domain.ErrInvalidInput},
		{"unknown", sentinel, sentinel},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapDBError(tc.in)
			if tc.target == nil {
				if got != nil {
					t.Fatalf("got %v", got)
				}
				return
			}
			if !errors.Is(got, tc.target) {
				t.Fatalf("got %v, want errors.Is(...,%v)", got, tc.target)
			}
		})
	}
}

func TestRequireAffected(t *testing.T) {
	if err := requireAffected(pgconn.NewCommandTag("UPDATE 0")); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("zero affected: %v", err)
	}
	if err := requireAffected(pgconn.NewCommandTag("UPDATE 1")); err != nil {
		t.Fatalf("one affected: %v", err)
	}
	if err := requireAffected(pgconn.NewCommandTag("DELETE 12")); err != nil {
		t.Fatalf("many affected: %v", err)
	}
}
