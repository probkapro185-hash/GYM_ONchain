package httphandler

import "testing"

func TestCSVValuesAreNumbersInsteadOfPointerAddresses(t *testing.T) {
	sessions, weight := 8, 72.5
	if got := csvSessions(&sessions); got != "8" {
		t.Fatalf("sessions=%q", got)
	}
	if got := csvSessions(nil); got != "Безлимит" {
		t.Fatalf("unlimited=%q", got)
	}
	if got := csvMeasurement(&weight); got != "72.5" {
		t.Fatalf("weight=%q", got)
	}
	if got := csvMeasurement(nil); got != "—" {
		t.Fatalf("missing measurement=%q", got)
	}
}
