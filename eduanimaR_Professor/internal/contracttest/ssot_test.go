package contracttest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatalf("repo root not found from %s", wd)
		}
		current = parent
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestOpenAPISsotExistsAndHasErrorShape(t *testing.T) {
	root := repoRoot(t)
	p := filepath.Join(root, "docs", "openapi.yaml")
	content := mustReadFile(t, p)

	if !strings.Contains(content, "openapi:") {
		t.Fatalf("openapi.yaml must include 'openapi:'")
	}
	if !strings.Contains(content, "components:") {
		t.Fatalf("openapi.yaml must include components")
	}
	if !strings.Contains(content, "ErrorResponse:") {
		t.Fatalf("openapi.yaml must define components.schemas.ErrorResponse")
	}
	if !strings.Contains(content, "request_id") {
		t.Fatalf("ErrorResponse should include request_id")
	}
	if strings.Contains(content, "paths: {}") {
		t.Fatalf("openapi.yaml must define non-empty paths (SSOT must be actionable)")
	}
	if !strings.Contains(content, "/v1/questions:") {
		t.Fatalf("openapi.yaml must define /v1/questions")
	}
}

func TestProtoSsotExistsAndHasLibrarianService(t *testing.T) {
	root := repoRoot(t)
	p := filepath.Join(root, "proto", "librarian", "v1", "librarian.proto")
	content := mustReadFile(t, p)

	if !strings.Contains(content, "syntax = \"proto3\"") {
		t.Fatalf("librarian.proto must declare proto3 syntax")
	}
	if !strings.Contains(content, "package librarian.v1") {
		t.Fatalf("librarian.proto must declare package librarian.v1")
	}
	if !strings.Contains(content, "service LibrarianService") {
		t.Fatalf("librarian.proto must define service LibrarianService")
	}
	if !strings.Contains(content, "rpc Reason") {
		t.Fatalf("librarian.proto must define Reason RPC")
	}
}
