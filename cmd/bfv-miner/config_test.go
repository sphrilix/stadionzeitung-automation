package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sz.json")
	err := os.WriteFile(path, []byte(`{
		"outDir": "Dateien",
		"fonts": {"regular": "fonts/calibri.ttf", "bold": "/abs/calibrib.ttf"},
		"teams": [{"file": "Spielplan-H1.pdf", "league": "0307PL23RO000007VS5489BUVT69HBT7-G"}]
	}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	c, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "Dateien"); c.OutDir != want {
		t.Errorf("OutDir = %q, want %q", c.OutDir, want)
	}
	if want := filepath.Join(dir, "fonts/calibri.ttf"); c.Fonts.Regular != want {
		t.Errorf("Fonts.Regular = %q, want %q", c.Fonts.Regular, want)
	}
	if c.Fonts.Bold != "/abs/calibrib.ttf" {
		t.Errorf("Fonts.Bold = %q, want it unchanged", c.Fonts.Bold)
	}
}

func TestLoadConfigRejectsIncompleteTeam(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sz.json")
	if err := os.WriteFile(path, []byte(`{"teams": [{"file": "Spielplan-H1.pdf"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(path); err == nil {
		t.Error("expected an error for a team without league")
	}
}

func TestLoadConfigRejectsUnknownKind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sz.json")
	err := os.WriteFile(path, []byte(`{"teams": [
		{"file": "Tabelle-H1.pdf", "kind": "tabelle", "league": "0307PL23RO000007VS5489BUVT69HBT7-G"},
		{"file": "x.pdf", "kind": "kaffee", "league": "0307PL23RO000007VS5489BUVT69HBT7-G"}]}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(path); err == nil || !strings.Contains(err.Error(), "team 2") {
		t.Errorf("err = %v, want an unknown kind error for team 2", err)
	}
}

func TestWriteFileAtomicKeepsOldFileOnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Spielplan-H1.pdf")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := writeFileAtomic(path, func(w io.Writer) error {
		io.WriteString(w, "partial")
		return errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected the write error")
	}
	if b, _ := os.ReadFile(path); string(b) != "old" {
		t.Errorf("file = %q, want the old content", b)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("leftover temp files: %v", entries)
	}

	if err := writeFileAtomic(path, func(w io.Writer) error { _, err := io.WriteString(w, "new"); return err }); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != "new" {
		t.Errorf("file = %q, want the new content", b)
	}
}
