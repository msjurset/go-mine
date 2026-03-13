package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/msjurset/golars"
)

func TestExportCSV(t *testing.T) {
	df := testDataFrame(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")

	em := NewExportModel()
	em.SetDataFrame(df)
	em.input.SetValue(path)
	em, _ = em.doExport()

	if em.err != nil {
		t.Fatalf("export error: %v", em.err)
	}
	if em.done != path {
		t.Errorf("expected done=%q, got %q", path, em.done)
	}

	// Verify file exists and can be read back
	loaded, err := golars.ReadCSV(path)
	if err != nil {
		t.Fatalf("failed to read exported CSV: %v", err)
	}
	if loaded.Height() != df.Height() {
		t.Errorf("expected %d rows, got %d", df.Height(), loaded.Height())
	}
}

func TestExportParquet(t *testing.T) {
	df := testDataFrame(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "out.parquet")

	em := NewExportModel()
	em.SetDataFrame(df)
	em.input.SetValue(path)
	em, _ = em.doExport()

	if em.err != nil {
		t.Fatalf("export error: %v", em.err)
	}

	loaded, err := golars.ReadParquet(path)
	if err != nil {
		t.Fatalf("failed to read exported Parquet: %v", err)
	}
	if loaded.Height() != df.Height() {
		t.Errorf("expected %d rows, got %d", df.Height(), loaded.Height())
	}
}

func TestExportJSON(t *testing.T) {
	df := testDataFrame(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	em := NewExportModel()
	em.SetDataFrame(df)
	em.input.SetValue(path)
	em, _ = em.doExport()

	if em.err != nil {
		t.Fatalf("export error: %v", em.err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("exported file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("exported JSON file is empty")
	}
}

func TestExportUnsupportedFormat(t *testing.T) {
	df := testDataFrame(t)
	em := NewExportModel()
	em.SetDataFrame(df)
	em.input.SetValue("/tmp/out.xlsx")
	em, _ = em.doExport()

	if em.err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestExportEmptyPath(t *testing.T) {
	df := testDataFrame(t)
	em := NewExportModel()
	em.SetDataFrame(df)
	em.input.SetValue("")
	em, _ = em.doExport()

	if em.err == nil {
		t.Error("expected error for empty path")
	}
}
