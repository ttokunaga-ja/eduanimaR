package contract

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
		if _, err := os.Stat(filepath.Join(current, "contracts")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatalf("repository root not found from %s", wd)
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
	p := filepath.Join(root, "contracts", "openapi", "professor.yaml")
	content := mustReadFile(t, p)

	if !strings.Contains(content, "openapi:") {
		t.Fatalf("professor.yaml must include 'openapi:'")
	}
	if !strings.Contains(content, "components:") {
		t.Fatalf("professor.yaml must include components")
	}
	if !strings.Contains(content, "ErrorResponse") {
		t.Fatalf("professor.yaml must define ErrorResponse schema")
	}
	if !strings.Contains(content, "request_id") {
		t.Fatalf("ErrorResponse should include request_id")
	}
}

func TestProtoSsotExistsAndHasLibrarianService(t *testing.T) {
	root := repoRoot(t)
	p := filepath.Join(root, "contracts", "proto", "librarian", "v1", "librarian.proto")
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
	if !strings.Contains(content, "rpc Think") {
		t.Fatalf("librarian.proto must define Think RPC")
	}
}
