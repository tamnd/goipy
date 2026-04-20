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
	matches, err := filepath.Glob("../internal/testdata/*.pyc")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("no fixtures; run internal/testdata/gen.sh")
	}
	for _, pyc := range matches {
		name := strings.TrimSuffix(filepath.Base(pyc), ".pyc")
		t.Run(name, func(t *testing.T) {
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
