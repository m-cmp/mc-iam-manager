package service

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPermissionSeedSourceMatchesExt(t *testing.T) {
	cases := []struct {
		source     string
		defaultExt string
		want       bool
	}{
		{"https://example.com/perm.csv", ".yaml", false},
		{"https://example.com/perm.csv?token=1", ".csv", true},
		{"/tmp/permission.yaml", ".yml", true},
		{"/tmp/permission.yml", ".yaml", true},
		{"/tmp/permission.csv", ".csv", true},
		{"/tmp/permission.yaml", ".csv", false},
	}
	for _, tc := range cases {
		got := permissionSeedSourceMatchesExt(tc.source, tc.defaultExt)
		if got != tc.want {
			t.Fatalf(
				"permissionSeedSourceMatchesExt(%q, %q)=%v, want %v",
				tc.source, tc.defaultExt, got, tc.want,
			)
		}
	}
}

func TestResolvePermissionSeedPathSharedEnvYAMLPreferred(t *testing.T) {
	yamlPath := filepath.Join(t.TempDir(), "custom-permission.yaml")
	t.Setenv("MC_WEB_CONSOLE_MENU_PERMISSIONS", yamlPath)

	svc := &MenuService{}
	got, cleanup, err := svc.resolvePermissionSeedPath("", "permission.yaml", ".yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleanup != "" {
		t.Fatalf("cleanup=%q, want empty", cleanup)
	}
	if got != yamlPath {
		t.Fatalf("got %q, want shared YAML path %q", got, yamlPath)
	}
}

func TestResolvePermissionSeedPathCSVEnvIgnoredForYAML(t *testing.T) {
	t.Setenv(
		"MC_WEB_CONSOLE_MENU_PERMISSIONS",
		"https://example.com/webconsole_menu_permissions.csv",
	)

	svc := &MenuService{}
	got, cleanup, err := svc.resolvePermissionSeedPath("", "permission.yaml", ".yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleanup != "" {
		t.Fatalf("cleanup=%q, want empty (no download)", cleanup)
	}
	if !strings.HasSuffix(got, filepath.Join("menu", "permission.yaml")) {
		t.Fatalf("got %q, want path ending with menu/permission.yaml", got)
	}
}

func TestResolvePermissionSeedPathCSVEnvUsedForCSV(t *testing.T) {
	csvPath := filepath.Join(t.TempDir(), "custom-permission.csv")
	t.Setenv("MC_WEB_CONSOLE_MENU_PERMISSIONS", csvPath)

	svc := &MenuService{}
	got, cleanup, err := svc.resolvePermissionSeedPath("", "permission.csv", ".csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleanup != "" {
		t.Fatalf("cleanup=%q, want empty", cleanup)
	}
	if got != csvPath {
		t.Fatalf("got %q, want CSV path %q", got, csvPath)
	}
}

func TestResolvePermissionSeedPathExplicitFilePathWins(t *testing.T) {
	t.Setenv(
		"MC_WEB_CONSOLE_MENU_PERMISSIONS",
		"/env/should/not/use/permission.yaml",
	)

	explicit := "/explicit/permission.yaml"
	svc := &MenuService{}
	got, cleanup, err := svc.resolvePermissionSeedPath(explicit, "permission.yaml", ".yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleanup != "" {
		t.Fatalf("cleanup=%q, want empty", cleanup)
	}
	if got != explicit {
		t.Fatalf("got %q, want explicit %q", got, explicit)
	}
}
