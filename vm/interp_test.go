package vm

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/goipy/marshal"
)

func TestFixtures(t *testing.T) {
	// Use absolute paths so that os.Chdir calls inside Python scripts
	// don't break subsequent subtests.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	matches, err := filepath.Glob("../internal/testdata/*.pyc")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("no fixtures; run internal/testdata/gen.sh")
	}
	// Convert to absolute paths before any subtest can change the CWD.
	absPaths := make([]string, len(matches))
	for j, m := range matches {
		abs, aerr := filepath.Abs(m)
		if aerr != nil {
			t.Fatal(aerr)
		}
		absPaths[j] = abs
	}
	for _, pyc := range absPaths {
		name := strings.TrimSuffix(filepath.Base(pyc), ".pyc")
		t.Run(name, func(t *testing.T) {
			// Restore working directory after each subtest in case the
			// Python script called os.chdir().
			defer os.Chdir(origDir) //nolint:errcheck
			code, err := marshal.LoadPyc(pyc)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			expected, err := os.ReadFile(strings.TrimSuffix(pyc, ".pyc") + ".expected.txt")
			if err != nil {
				t.Fatalf("expected: %v", err)
			}
			var buf bytes.Buffer
			i := New()
			i.Stdout = &buf
			i.SearchPath = []string{filepath.Dir(pyc)}
			if _, err := i.Run(code); err != nil {
				t.Fatalf("run: %v\noutput so far:\n%s", err, buf.String())
			}
			got := buf.String()
			if got != string(expected) {
				t.Errorf("output mismatch\n--- expected ---\n%s--- got ---\n%s", expected, got)
			}
		})
	}
}
