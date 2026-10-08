package config

import "testing"

func TestLowMemoryProfile(t *testing.T) {
	t.Setenv("BOT_TOKEN", "fake")
	t.Setenv("MAX_WORKERS", "5")
	t.Setenv("REDUCE_MEDIA", "false")
	for _, tc := range []struct {
		value   string
		workers int
		invalid bool
	}{
		{"true", 1, false}, {"false", 5, false}, {"typo", 0, true},
	} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("LOW_MEMORY", tc.value)
			cfg, err := Load()
			if (err != nil) != tc.invalid {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && (cfg.MaxWorkers != tc.workers || cfg.ReduceMedia) {
				t.Fatalf("wrong profile: %+v", cfg)
			}
		})
	}
}

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
