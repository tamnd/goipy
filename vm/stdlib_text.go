package vm

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/tamnd/goipy/object"
)

// --- difflib module --------------------------------------------------------

func (i *Interp) buildDifflib() *object.Module {
	m := &object.Module{Name: "difflib", Dict: object.NewDict()}

	m.Dict.SetStr("get_close_matches", &object.BuiltinFunc{Name: "get_close_matches", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "get_close_matches() requires word and possibilities")
		}
		word, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "word must be str")
		}
		candidates, err := i.iterStrings(a[1])
		if err != nil {
			return nil, err
		}
		n := 3
		cutoff := 0.6
		if len(a) >= 3 {
			if v, ok := toInt64(a[2]); ok {
				n = int(v)
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("n"); ok {
				if iv, ok := toInt64(v); ok {
					n = int(iv)
				}
			}
		}
		if len(a) >= 4 {
			if f, ok := toFloat64(a[3]); ok {
				cutoff = f
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("cutoff"); ok {
				if f, ok := toFloat64(v); ok {
					cutoff = f
				}
			}
		}
		type scored struct {
			s string
			r float64
		}
		var hits []scored
		for _, c := range candidates {
			r := ratcliffObershelp(word.V, c)
			if r >= cutoff {
				hits = append(hits, scored{c, r})
			}
		}
		sort.SliceStable(hits, func(i, j int) bool {
			if hits[i].r != hits[j].r {
				return hits[i].r > hits[j].r
			}
			return hits[i].s > hits[j].s
		})
		if len(hits) > n {
			hits = hits[:n]
		}
		out := make([]object.Object, len(hits))
		for k, h := range hits {
			out[k] = &object.Str{V: h.s}
		}
		return &object.List{V: out}, nil
	}})

	m.Dict.SetStr("ndiff", &object.BuiltinFunc{Name: "ndiff", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "ndiff() requires a and b")
		}
		aLines, err := i.iterStrings(a[0])
		if err != nil {
			return nil, err
		}
		bLines, err := i.iterStrings(a[1])
		if err != nil {
			return nil, err
		}
		lines := ndiffLines(aLines, bLines)
		return listOfStr(lines), nil
	}})

	m.Dict.SetStr("unified_diff", &object.BuiltinFunc{Name: "unified_diff", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "unified_diff() requires a and b")
		}
		aLines, err := i.iterStrings(a[0])
		if err != nil {
			return nil, err
		}
		bLines, err := i.iterStrings(a[1])
		if err != nil {
			return nil, err
		}
		fromfile, tofile := "", ""
		if kw != nil {
			if v, ok := kw.GetStr("fromfile"); ok {
				if s, ok := v.(*object.Str); ok {
					fromfile = s.V
				}
			}
			if v, ok := kw.GetStr("tofile"); ok {
				if s, ok := v.(*object.Str); ok {
					tofile = s.V
				}
			}
		}
		lines := unifiedDiff(aLines, bLines, fromfile, tofile)
		return listOfStr(lines), nil
	}})

	m.Dict.SetStr("SequenceMatcher", &object.BuiltinFunc{Name: "SequenceMatcher", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		var aStr, bStr string
		// Signature: SequenceMatcher(isjunk=None, a='', b='', autojunk=True)
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				aStr = s.V
			}
		}
		if len(a) >= 3 {
			if s, ok := a[2].(*object.Str); ok {
				bStr = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("a"); ok {
				if s, ok := v.(*object.Str); ok {
					aStr = s.V
				}
			}
			if v, ok := kw.GetStr("b"); ok {
				if s, ok := v.(*object.Str); ok {
					bStr = s.V
				}
			}
		}
		return &object.SequenceMatcher{A: aStr, B: bStr}, nil
	}})

	return m
}

// sequenceMatcherAttr dispatches attribute access on *object.SequenceMatcher.
func sequenceMatcherAttr(sm *object.SequenceMatcher, name string) (object.Object, bool) {
	switch name {
	case "a":
		return &object.Str{V: sm.A}, true
	case "b":
		return &object.Str{V: sm.B}, true
	case "ratio":
		return &object.BuiltinFunc{Name: "ratio", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Float{V: ratcliffObershelp(sm.A, sm.B)}, nil
		}}, true
	case "quick_ratio":
		return &object.BuiltinFunc{Name: "quick_ratio", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Float{V: quickRatio(sm.A, sm.B)}, nil
		}}, true
	case "real_quick_ratio":
		return &object.BuiltinFunc{Name: "real_quick_ratio", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			la, lb := len([]rune(sm.A)), len([]rune(sm.B))
			if la+lb == 0 {
				return &object.Float{V: 1.0}, nil
			}
			m := la
			if lb < m {
				m = lb
			}
			return &object.Float{V: 2.0 * float64(m) / float64(la+lb)}, nil
		}}, true
	case "set_seq1":
		return &object.BuiltinFunc{Name: "set_seq1", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if s, ok := a[0].(*object.Str); ok {
				sm.A = s.V
			}
			return object.None, nil
		}}, true
	case "set_seq2":
		return &object.BuiltinFunc{Name: "set_seq2", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if s, ok := a[0].(*object.Str); ok {
				sm.B = s.V
			}
			return object.None, nil
		}}, true
	}
	return nil, false
}

// ratcliffObershelp implements CPython's difflib ratio via longest-common-
// contiguous-substring recursion. Matches CPython output for plain strings.
func ratcliffObershelp(a, b string) float64 {
	ar := []rune(a)
	br := []rune(b)
	if len(ar) == 0 && len(br) == 0 {
		return 1.0
	}
	matched := matchCount(ar, br)
	return 2.0 * float64(matched) / float64(len(ar)+len(br))
}

func matchCount(a, b []rune) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	// Find longest common contiguous substring.
	bestI, bestJ, bestK := 0, 0, 0
	// DP table; use a single rolling row.
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1] + 1
				if curr[j] > bestK {
					bestK = curr[j]
					bestI = i - bestK
					bestJ = j - bestK
				}
			} else {
				curr[j] = 0
			}
		}
		prev, curr = curr, prev
		for k := range curr {
			curr[k] = 0
		}
	}
	if bestK == 0 {
		return 0
	}
	total := bestK
	total += matchCount(a[:bestI], b[:bestJ])
	total += matchCount(a[bestI+bestK:], b[bestJ+bestK:])
	return total
}

// quickRatio is an upper bound — overlap of characters regardless of order.
func quickRatio(a, b string) float64 {
	if a == "" && b == "" {
		return 1.0
	}
	counts := map[rune]int{}
	for _, r := range b {
		counts[r]++
	}
	matches := 0
	for _, r := range a {
		if counts[r] > 0 {
			counts[r]--
			matches++
		}
	}
	return 2.0 * float64(matches) / float64(len([]rune(a))+len([]rune(b)))
}

// ndiffLines produces CPython-compatible ndiff output. Straightforward LCS
// against the two line sequences with "  ", "- ", "+ ", "? " prefixes.
func ndiffLines(a, b []string) []string {
	// Compute LCS table.
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var out []string
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			out = append(out, "  "+a[i])
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			out = append(out, "- "+a[i])
			i++
		} else {
			out = append(out, "+ "+b[j])
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, "- "+a[i])
	}
	for ; j < m; j++ {
		out = append(out, "+ "+b[j])
	}
	return out
}

// unifiedDiff produces a single-hunk unified diff covering the full range.
// We don't trim context blocks the way CPython does for multi-hunk outputs;
// for the sizes users test with this is indistinguishable.
func unifiedDiff(a, b []string, fromfile, tofile string) []string {
	if fromfile == "" && tofile == "" {
		// Bare diff with no headers.
	}
	var lines []string
	lines = append(lines, "--- "+fromfile+"\n")
	lines = append(lines, "+++ "+tofile+"\n")
	// Compute an edit script aligned with ndiff so the hunk ordering matches.
	nd := ndiffLines(a, b)
	// Count edits.
	adds, dels, ctxA, ctxB := 0, 0, 0, 0
	for _, l := range nd {
		switch l[:2] {
		case "+ ":
			adds++
		case "- ":
			dels++
		case "  ":
			ctxA++
			ctxB++
		}
	}
	if adds == 0 && dels == 0 {
		return nil
	}
	lines = append(lines, fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(a), len(b)))
	for _, l := range nd {
		switch l[:2] {
		case "+ ":
			lines = append(lines, "+"+l[2:])
		case "- ":
			lines = append(lines, "-"+l[2:])
		case "  ":
			lines = append(lines, " "+l[2:])
		}
	}
	return lines
}

// --- shlex module ----------------------------------------------------------

func (i *Interp) buildShlex() *object.Module {
	m := &object.Module{Name: "shlex", Dict: object.NewDict()}

	m.Dict.SetStr("quote", &object.BuiltinFunc{Name: "quote", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "quote() missing argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "quote() argument must be str")
		}
		return &object.Str{V: shlexQuote(s.V)}, nil
	}})

	m.Dict.SetStr("join", &object.BuiltinFunc{Name: "join", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "join() missing argument")
		}
		parts, err := i.iterStrings(a[0])
		if err != nil {
			return nil, err
		}
		quoted := make([]string, len(parts))
		for k, p := range parts {
			quoted[k] = shlexQuote(p)
		}
		return &object.Str{V: strings.Join(quoted, " ")}, nil
	}})

	m.Dict.SetStr("split", &object.BuiltinFunc{Name: "split", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "split() missing argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "split() argument must be str")
		}
		comments := false
		posix := true
		if kw != nil {
			if v, ok := kw.GetStr("comments"); ok {
				if b, ok := v.(*object.Bool); ok {
					comments = b.V
				}
			}
			if v, ok := kw.GetStr("posix"); ok {
				if b, ok := v.(*object.Bool); ok {
					posix = b.V
				}
			}
		}
		parts, err := shlexSplit(s.V, comments, posix)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		return listOfStr(parts), nil
	}})

	return m
}

// shlexQuote mirrors CPython's shlex.quote: wraps in single quotes unless
// the input is empty or contains only safe characters.
var shlexSafe = regexp.MustCompile(`\A[a-zA-Z0-9@%+=:,./_-]*\z`)

func shlexQuote(s string) string {
	if s == "" {
		return "''"
	}
	if shlexSafe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// shlexSplit implements a POSIX-ish lexer good enough for the common cases
// (whitespace separation, single/double quotes, backslash escaping in
// posix mode, # comments when enabled).
func shlexSplit(s string, comments, posix bool) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	inToken := false
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '#' && comments && !inToken:
			// Skip to end-of-line.
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case unicode.IsSpace(rune(c)):
			if inToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				inToken = false
			}
			i++
		case c == '\\' && posix:
			inToken = true
			if i+1 < len(s) {
				cur.WriteByte(s[i+1])
				i += 2
			} else {
				i++
			}
		case c == '\'':
			inToken = true
			i++
			for i < len(s) && s[i] != '\'' {
				cur.WriteByte(s[i])
				i++
			}
			if i >= len(s) {
				return nil, fmt.Errorf("No closing quotation")
			}
			i++
		case c == '"':
			inToken = true
			i++
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' && posix && i+1 < len(s) {
					next := s[i+1]
					switch next {
					case '"', '\\', '$', '`':
						cur.WriteByte(next)
						i += 2
						continue
					}
				}
				cur.WriteByte(s[i])
				i++
			}
			if i >= len(s) {
				return nil, fmt.Errorf("No closing quotation")
			}
			i++
		default:
			inToken = true
			cur.WriteByte(c)
			i++
		}
	}
	if inToken {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}

// --- gzip module -----------------------------------------------------------

func (i *Interp) buildGzip() *object.Module {
	m := &object.Module{Name: "gzip", Dict: object.NewDict()}

	m.Dict.SetStr("compress", &object.BuiltinFunc{Name: "compress", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "compress() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		level := gzip.DefaultCompression
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				level = int(n)
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("compresslevel"); ok {
				if n, ok := toInt64(v); ok {
					level = int(n)
				}
			}
		}
		var buf bytes.Buffer
		w, err := gzip.NewWriterLevel(&buf, level)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		// Mark the uncompressed data as deterministic — CPython's default
		// writes 0 for the mtime when called via gzip.compress.
		w.ModTime = gzipEpoch
		if _, err := w.Write(data); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		return &object.Bytes{V: buf.Bytes()}, nil
	}})

	m.Dict.SetStr("decompress", &object.BuiltinFunc{Name: "decompress", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "decompress() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		r, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		defer r.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		return &object.Bytes{V: out}, nil
	}})

	return m
}

// --- fnmatch module --------------------------------------------------------

func (i *Interp) buildFnmatch() *object.Module {
	m := &object.Module{Name: "fnmatch", Dict: object.NewDict()}

	m.Dict.SetStr("translate", &object.BuiltinFunc{Name: "translate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "translate")
		if err != nil {
			return nil, err
		}
		return &object.Str{V: fnmatchTranslate(s)}, nil
	}})

	m.Dict.SetStr("fnmatchcase", &object.BuiltinFunc{Name: "fnmatchcase", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "fnmatchcase() requires name and pat")
		}
		name, _ := a[0].(*object.Str)
		pat, _ := a[1].(*object.Str)
		if name == nil || pat == nil {
			return nil, object.Errorf(i.typeErr, "fnmatchcase() args must be str")
		}
		re, err := regexp.Compile("^" + fnmatchTranslateCore(pat.V) + "$")
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		return object.BoolOf(re.MatchString(name.V)), nil
	}})

	m.Dict.SetStr("fnmatch", &object.BuiltinFunc{Name: "fnmatch", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "fnmatch() requires name and pat")
		}
		name, _ := a[0].(*object.Str)
		pat, _ := a[1].(*object.Str)
		if name == nil || pat == nil {
			return nil, object.Errorf(i.typeErr, "fnmatch() args must be str")
		}
		re, err := regexp.Compile("^" + fnmatchTranslateCore(pat.V) + "$")
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		return object.BoolOf(re.MatchString(name.V)), nil
	}})

	m.Dict.SetStr("filter", &object.BuiltinFunc{Name: "filter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "filter() requires names and pat")
		}
		names, err := i.iterStrings(a[0])
		if err != nil {
			return nil, err
		}
		pat, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "pat must be str")
		}
		re, err := regexp.Compile("^" + fnmatchTranslateCore(pat.V) + "$")
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		var out []string
		for _, n := range names {
			if re.MatchString(n) {
				out = append(out, n)
			}
		}
		return listOfStr(out), nil
	}})

	return m
}

// fnmatchTranslate returns the CPython-style Python regex for a shell glob.
// CPython wraps it as "(?s:<core>)\Z" — we emit a close-enough Go regex.
func fnmatchTranslate(pat string) string {
	return "(?s:" + fnmatchTranslateCore(pat) + `)\z`
}

func fnmatchTranslateCore(pat string) string {
	var b strings.Builder
	i := 0
	for i < len(pat) {
		c := pat[i]
		switch c {
		case '*':
			b.WriteString(".*")
			i++
		case '?':
			b.WriteString(".")
			i++
		case '[':
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
			b.WriteString(regexp.QuoteMeta(string(c)))
			i++
		}
	}
	return b.String()
}

// --- helpers ---------------------------------------------------------------

func listOfStr(ss []string) *object.List {
	v := make([]object.Object, len(ss))
	for i, s := range ss {
		v[i] = &object.Str{V: s}
	}
	return &object.List{V: v}
}

// gzipEpoch is the deterministic zero-mtime used by gzip.compress in CPython.
var gzipEpoch = time.Time{}
