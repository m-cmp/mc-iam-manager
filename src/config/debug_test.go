package config

import "testing"

func TestDebugEnabled(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"unset defaults to false", "", false},
		{"true", "true", true},
		{"numeric one", "1", true},
		{"false", "false", false},
		{"unparseable falls back to false", "yes", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MC_IAM_MANAGER_DEBUG", tc.raw)
			if got := DebugEnabled(); got != tc.want {
				t.Fatalf("MC_IAM_MANAGER_DEBUG=%q: got %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}
