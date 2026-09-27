package main

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sfedu-crm/pkg/hash"
)

type bootstrapRow struct {
	values []any
	err    error
}

func (r bootstrapRow) Scan(dst ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, v := range r.values {
		reflect.ValueOf(dst[i]).Elem().Set(reflect.ValueOf(v))
	}
	return nil
}

type bootstrapFake struct {
	rows  []bootstrapRow
	query string
	args  []any
}

func (f *bootstrapFake) QueryRow(context.Context, string, ...any) pgx.Row {
	r := f.rows[0]
	f.rows = f.rows[1:]
	return r
}
func (f *bootstrapFake) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	f.query = q
	f.args = args
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func TestBootstrapReusesSeedForCustomEmail(t *testing.T) {
	db := &bootstrapFake{rows: []bootstrapRow{{err: pgx.ErrNoRows}, {values: []any{int64(1)}}}}
	h := hash.NewBcrypt(4)
	if err := bootstrapAdminRecord(context.Background(), db, h, "owner@gmail.com", "test-password"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(db.query, "UPDATE users") || db.args[0] != "owner@gmail.com" || db.args[2] != int64(1) {
		t.Fatalf("must initialize the seed without inserting its phone again: %+v", db)
	}
	if !h.Compare(db.args[1].(string), "test-password") {
		t.Fatal("password was not hashed")
	}
}

func TestBootstrapDoesNotResetAnInitializedAccount(t *testing.T) {
	for _, rows := range [][]bootstrapRow{
		{{values: []any{int64(1), "existing-hash", "admin"}}},
		{{err: pgx.ErrNoRows}, {err: pgx.ErrNoRows}, {values: []any{true}}},
	} {
		db := &bootstrapFake{rows: rows}
		if err := bootstrapAdminRecord(context.Background(), db, hash.NewBcrypt(4), "owner@gmail.com", "test-password"); err != nil {
			t.Fatal(err)
		}
		if db.query != "" {
			t.Fatal("initialized account was changed")
		}
	}
}

func TestBootstrapRejectsEmailOwnedByNonAdmin(t *testing.T) {
	db := &bootstrapFake{rows: []bootstrapRow{{values: []any{int64(2), "existing-hash", "client"}}}}
	if err := bootstrapAdminRecord(context.Background(), db, hash.NewBcrypt(4), "client@gmail.com", "test-password"); err == nil {
		t.Fatal("must reject non-admin email")
	}
	if db.query != "" {
		t.Fatal("non-admin was changed")
	}
}
