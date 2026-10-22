package config

import "testing"

func TestSwaggerEnabled(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"unset defaults to true", "", true},
		{"true", "true", true},
		{"false", "false", false},
		{"numeric zero", "0", false},
		{"unparseable falls back to true", "no", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MC_IAM_MANAGER_SWAGGER_ENABLED", tc.raw)
			if got := SwaggerEnabled(); got != tc.want {
				t.Fatalf("MC_IAM_MANAGER_SWAGGER_ENABLED=%q: got %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}
