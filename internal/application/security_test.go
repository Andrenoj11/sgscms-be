package application

import "testing"

func TestPasswordRoundTrip(t *testing.T) {
	h, e := HashPassword("correct horse battery staple")
	if e != nil {
		t.Fatal(e)
	}
	if !VerifyPassword(h, "correct horse battery staple") {
		t.Fatal("valid password rejected")
	}
	if VerifyPassword(h, "wrong password") {
		t.Fatal("invalid password accepted")
	}
}
func TestSlug(t *testing.T) {
	if got := Slug("  Legal Update: July 2026! "); got != "legal-update-july-2026" {
		t.Fatalf("got %q", got)
	}
}
