package bot

import "testing"

func TestDonationParser(t *testing.T) {
	for _, args := range []string{"", "5", "5 pedro", "0 @pedro", "-5 @pedro", "+5 @pedro", "1.5 @pedro", "5 @pedro extra", "99999999999999999999 @pedro", "5 @", "5 @pedro!"} {
		if _, _, ok := parseDonation(args); ok {
			t.Fatal("accepted invalid donation", args)
		}
	}
	if amount, user, ok := parseDonation("5 @pedro"); !ok || amount != 5 || user != "@pedro" {
		t.Fatal(amount, user, ok)
	}
}
