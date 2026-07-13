package service

import (
	"testing"
)

func TestNormalizeAndValidateMenuResourceDefaults(t *testing.T) {
	viewType, frameworkService, path, err := normalizeAndValidateMenuResource("", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if viewType != defaultMenuViewType {
		t.Fatalf("viewType = %q, want %q", viewType, defaultMenuViewType)
	}
	if frameworkService != defaultMenuFrameworkService {
		t.Fatalf("frameworkService = %q, want %q", frameworkService, defaultMenuFrameworkService)
	}
	if path != "" {
		t.Fatalf("path = %q, want empty", path)
	}
}

func TestValidateMenuResourceIframeRequiresFramework(t *testing.T) {
	_, _, _, err := normalizeAndValidateMenuResource("iframe", "", "/")
	if err != ErrFrameworkServiceRequired {
		t.Fatalf("err = %v, want %v", err, ErrFrameworkServiceRequired)
	}
}

func TestValidateMenuResourceInvalidViewType(t *testing.T) {
	_, _, _, err := normalizeAndValidateMenuResource("external", "mc-web-console-front", "")
	if err != ErrInvalidViewType {
		t.Fatalf("err = %v, want %v", err, ErrInvalidViewType)
	}
}

func TestValidateMenuResourcePathTooLong(t *testing.T) {
	longPath := make([]byte, maxMenuPathLength+1)
	for i := range longPath {
		longPath[i] = 'a'
	}
	_, _, _, err := normalizeAndValidateMenuResource("local", "mc-web-console-front", string(longPath))
	if err != ErrPathTooLong {
		t.Fatalf("err = %v, want %v", err, ErrPathTooLong)
	}
}
