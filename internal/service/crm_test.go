package service

import "testing"

func TestValidRange(t *testing.T) {
	v := 80.5
	if !validRange(&v, 20, 500) {
		t.Fatal("expected value inside range")
	}
	bad := 700.0
	if validRange(&bad, 20, 500) {
		t.Fatal("expected out-of-range value to fail")
	}
	if !validRange(nil, 20, 500) {
		t.Fatal("nil optional metric should be valid")
	}
}
