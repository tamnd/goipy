package vm

import (
	"os"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// linecacheEntry holds the split lines for one cached file.
type linecacheEntry struct {
	lines []string // each line includes its trailing \n (last line may not)
	mtime int64    // file mtime at cache time; -1 for synthetic entries
	size  int64    // file size at cache time; -1 for synthetic entries
}

var (
	linecacheMu    sync.Mutex
	linecacheStore = map[string]*linecacheEntry{}
)

func (i *Interp) buildLinecache() *object.Module {
	m := &object.Module{Name: "linecache", Dict: object.NewDict()}

	// getline(filename, lineno, module_globals=None)
	m.Dict.SetStr("getline", &object.BuiltinFunc{Name: "getline", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "getline() requires filename and lineno")
		}
		fname, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "getline() filename must be str")
		}
		lno, ok2 := a[1].(*object.Int)
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "getline() lineno must be int")
		}
		line := linecacheGetLine(fname.V, int(lno.Int64()))
		return &object.Str{V: line}, nil
	}})

	// clearcache()
	m.Dict.SetStr("clearcache", &object.BuiltinFunc{Name: "clearcache", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		linecacheMu.Lock()
		linecacheStore = map[string]*linecacheEntry{}
		linecacheMu.Unlock()
		return object.None, nil
	}})

	// checkcache(filename=None)
	m.Dict.SetStr("checkcache", &object.BuiltinFunc{Name: "checkcache", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		var target string
		if len(a) > 0 {
			if s, ok := a[0].(*object.Str); ok {
				target = s.V
			}
		}
		linecacheCheckCache(target)
		return object.None, nil
	}})

	// lazycache(filename, module_globals)
	// Registers a placeholder so getline() knows to load the file on demand.
	// Since we always load from disk on cache miss, this is a no-op for real files;
	// we store a sentinel so the filename is "known" (mtime=-1 means synthetic).
	m.Dict.SetStr("lazycache", &object.BuiltinFunc{Name: "lazycache", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "lazycache() requires filename")
		}
		fname, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "lazycache() filename must be str")
		}
		linecacheMu.Lock()
		// Only register if not already cached.
		if _, exists := linecacheStore[fname.V]; !exists {
			linecacheStore[fname.V] = nil // nil sentinel = lazy, load on first getline
		}
		linecacheMu.Unlock()
		return object.None, nil
	}})

	return m
}

// linecacheGetLine returns the requested 1-indexed line from filename.
// Returns "" on any error (missing file, bad lineno).
func linecacheGetLine(filename string, lineno int) string {
	if lineno < 1 {
		return ""
	}
	lines := linecacheLoad(filename)
	if lineno > len(lines) {
		return ""
	}
	return lines[lineno-1]
}

// linecacheLoad returns the cached lines for filename, loading from disk if needed.
func linecacheLoad(filename string) []string {
	if filename == "" {
		return nil
	}
	linecacheMu.Lock()
	entry, exists := linecacheStore[filename]
	linecacheMu.Unlock()

	if exists && entry != nil {
		return entry.lines
	}

	// Load from disk.
	info, err := os.Stat(filename)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}

	lines := splitLines(string(data))
	e := &linecacheEntry{
		lines: lines,
		mtime: info.ModTime().UnixNano(),
		size:  info.Size(),
	}
	linecacheMu.Lock()
	linecacheStore[filename] = e
	linecacheMu.Unlock()
	return lines
}

// splitLines splits text into lines, each retaining its trailing \n.
// Matches CPython linecache.updatecache: empty files produce ["\n"],
// and the last line always has \n appended if missing.
func splitLines(text string) []string {
	if text == "" {
		// CPython: `if not lines: lines = ['\n']`
		return []string{"\n"}
	}
	raw := strings.Split(text, "\n")
	// strings.Split("a\nb\n", "\n") → ["a","b",""] — drop trailing empty.
	out := make([]string, 0, len(raw))
	for idx, part := range raw {
		isLast := idx == len(raw)-1
		if isLast && part == "" {
			break
		}
		out = append(out, part+"\n")
	}
	if len(out) == 0 {
		return []string{"\n"}
	}
	return out
}

// linecacheCheckCache validates cached entries against disk.
// If filename is non-empty, only that entry is checked; otherwise all entries.
func linecacheCheckCache(filename string) {
	linecacheMu.Lock()
	defer linecacheMu.Unlock()

	check := func(name string, entry *linecacheEntry) {
		if entry == nil {
			// lazy sentinel — leave as-is (will load on demand)
			return
		}
		if entry.mtime < 0 {
			// synthetic entry — keep forever
			return
		}
		info, err := os.Stat(name)
		if err != nil || info.ModTime().UnixNano() != entry.mtime || info.Size() != entry.size {
			// stale or gone — evict
			delete(linecacheStore, name)
		}
	}

	if filename != "" {
		if entry, ok := linecacheStore[filename]; ok {
			check(filename, entry)
		}
	} else {
		for name, entry := range linecacheStore {
			check(name, entry)
		}
	}
}
