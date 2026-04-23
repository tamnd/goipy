package vm

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildGlob() *object.Module {
	m := &object.Module{Name: "glob", Dict: object.NewDict()}

	// escape(pathname) — escape special glob chars using [X] notation.
	m.Dict.SetStr("escape", &object.BuiltinFunc{Name: "escape", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "escape() requires 1 argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "escape() argument must be str")
		}
		return &object.Str{V: globEscape(s.V)}, nil
	}})

	// translate(pathname, *, recursive=False, include_hidden=False, seps=None) — Python 3.12+
	// Returns a regex string matching the same paths as the pattern.
	m.Dict.SetStr("translate", &object.BuiltinFunc{Name: "translate", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "translate() requires 1 argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "translate() argument must be str")
		}
		recursive := false
		includeHidden := false
		if kw != nil {
			if v, ok2 := kw.GetStr("recursive"); ok2 {
				recursive = object.Truthy(v)
			}
			if v, ok2 := kw.GetStr("include_hidden"); ok2 {
				includeHidden = object.Truthy(v)
			}
		}
		return &object.Str{V: globTranslate(s.V, recursive, includeHidden)}, nil
	}})

	// parseGlobArgs extracts (pattern, root_dir, recursive, include_hidden) from a/kw.
	parseGlobArgs := func(a []object.Object, kw *object.Dict) (pattern, rootDir string, recursive, includeHidden bool, err error) {
		if len(a) < 1 {
			err = object.Errorf(i.typeErr, "glob() requires 1 argument")
			return
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			err = object.Errorf(i.typeErr, "glob() pathname must be str")
			return
		}
		pattern = s.V
		if kw != nil {
			if v, ok2 := kw.GetStr("root_dir"); ok2 && v != object.None {
				if rs, ok3 := v.(*object.Str); ok3 {
					rootDir = rs.V
				}
			}
			if v, ok2 := kw.GetStr("recursive"); ok2 {
				recursive = object.Truthy(v)
			}
			if v, ok2 := kw.GetStr("include_hidden"); ok2 {
				includeHidden = object.Truthy(v)
			}
		}
		return
	}

	// glob(pathname, *, root_dir=None, dir_fd=None, recursive=False, include_hidden=False)
	m.Dict.SetStr("glob", &object.BuiltinFunc{Name: "glob", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		pattern, rootDir, recursive, includeHidden, err := parseGlobArgs(a, kw)
		if err != nil {
			return nil, err
		}
		results := globRun(pattern, rootDir, recursive, includeHidden)
		out := make([]object.Object, len(results))
		for idx, r := range results {
			out[idx] = &object.Str{V: r}
		}
		return &object.List{V: out}, nil
	}})

	// iglob(pathname, *, root_dir=None, dir_fd=None, recursive=False, include_hidden=False)
	// Returns an iterator (we return a list iterator via *object.Iter).
	m.Dict.SetStr("iglob", &object.BuiltinFunc{Name: "iglob", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		pattern, rootDir, recursive, includeHidden, err := parseGlobArgs(a, kw)
		if err != nil {
			return nil, err
		}
		results := globRun(pattern, rootDir, recursive, includeHidden)
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(results) {
				return nil, false, nil
			}
			v := &object.Str{V: results[idx]}
			idx++
			return v, true, nil
		}}, nil
	}})

	return m
}

// globEscape escapes special glob characters using [X] notation (CPython style).
// * → [*]   ? → [?]   [ → [[]
func globEscape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, c := range s {
		switch c {
		case '*':
			b.WriteString("[*]")
		case '?':
			b.WriteString("[?]")
		case '[':
			b.WriteString("[[]")
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

// globTranslate converts a glob pattern to a regex string.
// Matches CPython's glob.translate() in Python 3.12+.
func globTranslate(pattern string, recursive, includeHidden bool) string {
	return globPatternToRegex(pattern, recursive, includeHidden)
}

// globPatternToRegex converts one glob pattern string to a full regex.
func globPatternToRegex(pat string, recursive, includeHidden bool) string {
	var b strings.Builder
	b.WriteString("(?s:")
	globWriteRegex(&b, pat, recursive, includeHidden)
	b.WriteString(`)\z`)
	return b.String()
}

// globWriteRegex writes the inner regex for a glob pattern segment.
func globWriteRegex(b *strings.Builder, pat string, recursive, includeHidden bool) {
	i := 0
	for i < len(pat) {
		c := pat[i]
		switch {
		case c == '/' :
			b.WriteString(`\/`)
			i++
		case c == '*' && i+1 < len(pat) && pat[i+1] == '*' && recursive:
			// ** — match any path (including separators), excluding hidden dirs if !includeHidden
			i += 2
			// consume trailing /
			if i < len(pat) && pat[i] == '/' {
				i++
				if !includeHidden {
					b.WriteString(`(?:[^./][^/]*/)*`)
				} else {
					b.WriteString(`(?:[^/]*/)*`)
				}
			} else {
				if !includeHidden {
					b.WriteString(`(?:(?:[^./][^/]*/)*[^./][^/]*)?`)
				} else {
					b.WriteString(`(?:[^/]+(?:/[^/]+)*)?`)
				}
			}
		case c == '*':
			i++
			if !includeHidden {
				b.WriteString(`[^./][^/]*`)
			} else {
				b.WriteString(`[^/]*`)
			}
		case c == '?':
			i++
			if !includeHidden {
				b.WriteString(`[^/.]`)
			} else {
				b.WriteString(`[^/]`)
			}
		case c == '[':
			end := strings.IndexByte(pat[i+1:], ']')
			if end < 0 {
				b.WriteString(`\[`)
				i++
				continue
			}
			class := pat[i+1 : i+1+end]
			i += 2 + end
			if len(class) > 0 && class[0] == '!' {
				class = "^" + class[1:]
			}
			b.WriteByte('[')
			b.WriteString(class)
			b.WriteByte(']')
		default:
			b.WriteString(regexp.QuoteMeta(string(rune(c))))
			i++
		}
	}
}

// globHasMagic reports whether a pattern contains glob magic characters.
func globHasMagic(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// globIsDoubleStar reports whether a path component is exactly "**".
func globIsDoubleStar(s string) bool { return s == "**" }

// globRun executes a glob pattern and returns sorted results.
// If rootDir is non-empty, paths are relative to rootDir.
func globRun(pattern, rootDir string, recursive, includeHidden bool) []string {
	// Detect trailing slash (directories only).
	dirOnly := strings.HasSuffix(pattern, "/")
	if dirOnly {
		pattern = strings.TrimRight(pattern, "/")
	}

	var results []string

	// Handle absolute patterns — collect from filesystem root, return absolute paths.
	if filepath.IsAbs(pattern) {
		base := string(os.PathSeparator)
		rel := strings.TrimPrefix(pattern, string(os.PathSeparator))
		globCollect(base, rel, recursive, includeHidden, dirOnly, &results)
		// results already have "/" prefix from globJoin
	} else if rootDir != "" {
		// root_dir: collect from rootDir on disk but build display paths relative to ".".
		var absResults []string
		globCollect(rootDir, pattern, recursive, includeHidden, dirOnly, &absResults)
		// Strip rootDir prefix to make paths relative.
		prefix := strings.TrimRight(rootDir, "/") + "/"
		for _, r := range absResults {
			rel := strings.TrimPrefix(r, prefix)
			results = append(results, rel)
		}
	} else {
		globCollect(".", pattern, recursive, includeHidden, dirOnly, &results)
	}

	sort.Strings(results)

	// Restore trailing slash for directory-only results.
	if dirOnly {
		for idx := range results {
			results[idx] += "/"
		}
	}
	return results
}

// globCollect recursively resolves one pattern against a base directory.
// All results are appended to *out as paths relative to the original base
// (i.e., without the rootDir prefix).
func globCollect(base, pattern string, recursive, includeHidden, dirOnly bool, out *[]string) {
	if pattern == "" {
		// Trailing-slash case: base itself must be a directory.
		if info, err := os.Lstat(base); err == nil && info.IsDir() {
			*out = append(*out, base)
		}
		return
	}

	// Split off the first path component.
	sepIdx := strings.IndexByte(pattern, '/')
	var head, tail string
	if sepIdx < 0 {
		head = pattern
	} else {
		head = pattern[:sepIdx]
		tail = pattern[sepIdx+1:]
	}

	isLast := sepIdx < 0

	// Handle ** component.
	if globIsDoubleStar(head) {
		globDoubleStarCollect(base, tail, recursive, includeHidden, dirOnly, isLast, out)
		return
	}

	if !globHasMagic(head) {
		// Exact component — no matching needed.
		path := globJoin(base, head)
		if isLast {
			info, err := os.Lstat(path)
			if err != nil {
				return
			}
			if dirOnly && !info.IsDir() {
				return
			}
			*out = append(*out, path)
		} else {
			globCollect(path, tail, recursive, includeHidden, dirOnly, out)
		}
		return
	}

	// Wildcard component — list entries and match.
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}

	for _, e := range entries {
		name := e.Name()

		// Hidden-file filtering: skip entries starting with '.' if the
		// pattern component doesn't start with '.' and include_hidden is off.
		if !includeHidden && strings.HasPrefix(name, ".") && !strings.HasPrefix(head, ".") {
			continue
		}

		if !globMatchComponent(head, name) {
			continue
		}

		path := globJoin(base, name)

		if isLast {
			if dirOnly {
				info, err := os.Lstat(path)
				if err != nil || !info.IsDir() {
					continue
				}
			}
			*out = append(*out, path)
		} else {
			// Recurse only into directories (or symlinks to directories).
			info, err := os.Lstat(path)
			if err != nil {
				continue
			}
			isDir := info.IsDir()
			if !isDir && info.Mode()&os.ModeSymlink != 0 {
				if target, err := os.Stat(path); err == nil {
					isDir = target.IsDir()
				}
			}
			if isDir {
				globCollect(path, tail, recursive, includeHidden, dirOnly, out)
			}
		}
	}
}

// globDoubleStarCollect handles the ** component.
// With recursive=True: matches zero or more directory levels.
// With recursive=False: ** acts like * (matches exactly one dir level).
func globDoubleStarCollect(base, tail string, recursive, includeHidden, dirOnly, isLast bool, out *[]string) {
	if !recursive {
		// Non-recursive: ** acts as * — match exactly one directory level.
		entries, err := os.ReadDir(base)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if !includeHidden && strings.HasPrefix(name, ".") {
				continue
			}
			path := globJoin(base, name)
			if tail == "" {
				*out = append(*out, path)
			} else {
				globCollect(path, tail, recursive, includeHidden, dirOnly, out)
			}
		}
		return
	}

	// recursive=True: ** matches zero or more path levels.
	// Zero levels: apply tail pattern directly to base.
	if tail != "" {
		globCollect(base, tail, recursive, includeHidden, dirOnly, out)
	} else {
		// ** alone at end: yield all non-hidden files and dirs recursively.
		globStarStarAll(base, includeHidden, out)
		return
	}

	// One or more levels: walk into subdirectories.
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !includeHidden && strings.HasPrefix(name, ".") {
			continue
		}
		subPath := globJoin(base, name)
		// Recurse into subdirectory with ** still active.
		var subPattern string
		if tail != "" {
			subPattern = "**/" + tail
		} else {
			subPattern = "**"
		}
		globCollect(subPath, subPattern, recursive, includeHidden, dirOnly, out)
	}
}

// globStarStarAll yields all non-hidden entries (files and dirs) under root recursively.
// Used for the pattern '**' alone with recursive=True.
func globStarStarAll(root string, includeHidden bool, out *[]string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !includeHidden && strings.HasPrefix(name, ".") {
			continue
		}
		path := globJoin(root, name)
		*out = append(*out, path)
		if e.IsDir() {
			globStarStarAll(path, includeHidden, out)
		}
	}
}

// globMatchComponent matches a single path component name against a pattern.
// Uses the fnmatch algorithm (but only for single component, no '/').
func globMatchComponent(pattern, name string) bool {
	re, err := regexp.Compile("^" + fnmatchTranslateCore(pattern) + "$")
	if err != nil {
		return false
	}
	return re.MatchString(name)
}

// globJoin joins base and name, handling the "." base case so that
// results like "./a.txt" come out as "a.txt".
func globJoin(base, name string) string {
	if base == "." {
		return name
	}
	if base == "/" {
		return "/" + name
	}
	return base + "/" + name
}

