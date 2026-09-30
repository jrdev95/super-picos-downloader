package config

import "testing"

func TestLoadMediaReduction(t *testing.T) {
	for _, tc := range []struct {
		value         string
		want, invalid bool
	}{{"", false, false}, {"false", false, false}, {"true", true, false}, {" true ", true, false}, {"typo", false, true}} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("REDUCE_MEDIA", tc.value)
			got, err := loadMediaReduction()
			if got != tc.want || (err != nil) != tc.invalid {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
}
