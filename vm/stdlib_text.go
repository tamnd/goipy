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

	// IS_LINE_JUNK: blank or comment-only line
	m.Dict.SetStr("IS_LINE_JUNK", &object.BuiltinFunc{Name: "IS_LINE_JUNK", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, ok := a[0].(*object.Str)
		if !ok {
			return object.BoolOf(false), nil
		}
		stripped := strings.TrimRight(s.V, " \t\r\n")
		return object.BoolOf(stripped == "" || stripped == "#"), nil
	}})

	// IS_CHARACTER_JUNK: space or tab
	m.Dict.SetStr("IS_CHARACTER_JUNK", &object.BuiltinFunc{Name: "IS_CHARACTER_JUNK", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, ok := a[0].(*object.Str)
		if !ok {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(s.V == " " || s.V == "\t"), nil
	}})

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
		fromfile, tofile, fromdate, todate := "", "", "", ""
		n := 3
		lineterm := "\n"
		if kw != nil {
			if v, ok := kw.GetStr("fromfile"); ok {
				if s, ok := v.(*object.Str); ok { fromfile = s.V }
			}
			if v, ok := kw.GetStr("tofile"); ok {
				if s, ok := v.(*object.Str); ok { tofile = s.V }
			}
			if v, ok := kw.GetStr("fromfiledate"); ok {
				if s, ok := v.(*object.Str); ok { fromdate = s.V }
			}
			if v, ok := kw.GetStr("tofiledate"); ok {
				if s, ok := v.(*object.Str); ok { todate = s.V }
			}
			if v, ok := kw.GetStr("n"); ok {
				if iv, ok2 := toInt64(v); ok2 { n = int(iv) }
			}
			if v, ok := kw.GetStr("lineterm"); ok {
				if s, ok := v.(*object.Str); ok { lineterm = s.V }
			}
		}
		lines := unifiedDiff(aLines, bLines, fromfile, tofile, fromdate, todate, n, lineterm)
		return listOfStr(lines), nil
	}})

	m.Dict.SetStr("context_diff", &object.BuiltinFunc{Name: "context_diff", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "context_diff() requires a and b")
		}
		aLines, err := i.iterStrings(a[0])
		if err != nil {
			return nil, err
		}
		bLines, err := i.iterStrings(a[1])
		if err != nil {
			return nil, err
		}
		fromfile, tofile, fromdate, todate := "", "", "", ""
		n := 3
		lineterm := "\n"
		if kw != nil {
			if v, ok := kw.GetStr("fromfile"); ok {
				if s, ok := v.(*object.Str); ok { fromfile = s.V }
			}
			if v, ok := kw.GetStr("tofile"); ok {
				if s, ok := v.(*object.Str); ok { tofile = s.V }
			}
			if v, ok := kw.GetStr("fromfiledate"); ok {
				if s, ok := v.(*object.Str); ok { fromdate = s.V }
			}
			if v, ok := kw.GetStr("tofiledate"); ok {
				if s, ok := v.(*object.Str); ok { todate = s.V }
			}
			if v, ok := kw.GetStr("n"); ok {
				if iv, ok2 := toInt64(v); ok2 { n = int(iv) }
			}
			if v, ok := kw.GetStr("lineterm"); ok {
				if s, ok := v.(*object.Str); ok { lineterm = s.V }
			}
		}
		lines := contextDiff(aLines, bLines, fromfile, tofile, fromdate, todate, n, lineterm)
		return listOfStr(lines), nil
	}})

	m.Dict.SetStr("restore", &object.BuiltinFunc{Name: "restore", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "restore() requires sequence and which")
		}
		lines, err := i.iterStrings(a[0])
		if err != nil {
			return nil, err
		}
		which, ok := toInt64(a[1])
		if !ok {
			return nil, object.Errorf(i.typeErr, "restore() which must be 1 or 2")
		}
		var prefix string
		if which == 1 {
			prefix = "- "
		} else {
			prefix = "+ "
		}
		var out []string
		for _, l := range lines {
			if strings.HasPrefix(l, "  ") {
				out = append(out, l[2:])
			} else if strings.HasPrefix(l, prefix) {
				out = append(out, l[2:])
			}
		}
		return listOfStr(out), nil
	}})

	// Differ class
	differCls := i.makeDifferClass()
	m.Dict.SetStr("Differ", differCls)

	m.Dict.SetStr("SequenceMatcher", &object.BuiltinFunc{Name: "SequenceMatcher", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		sm := &object.SequenceMatcher{}
		// Signature: SequenceMatcher(isjunk=None, a='', b='', autojunk=True)
		// args[0] = isjunk (ignored), args[1] = a, args[2] = b
		setSeq := func(idx int, kwName string) {
			var val object.Object
			if len(a) > idx {
				val = a[idx]
			} else if kw != nil {
				if v, ok := kw.GetStr(kwName); ok {
					val = v
				}
			}
			if val == nil {
				return
			}
			switch v := val.(type) {
			case *object.Str:
				if idx == 1 { sm.A = v.V } else { sm.B = v.V }
			case *object.List:
				if idx == 1 { sm.SeqA = v.V } else { sm.SeqB = v.V }
			case *object.Tuple:
				if idx == 1 { sm.SeqA = v.V } else { sm.SeqB = v.V }
			}
		}
		setSeq(1, "a")
		setSeq(2, "b")
		return sm, nil
	}})

	return m
}

// makeDifferClass returns a callable that creates Differ instances.
func (i *Interp) makeDifferClass() *object.BuiltinFunc {
	return &object.BuiltinFunc{Name: "Differ", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		inst := &object.Instance{Dict: object.NewDict()}
		inst.Dict.SetStr("compare", &object.BuiltinFunc{Name: "compare", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "compare() requires a and b")
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
		return inst, nil
	}}
}

// smSeqLen returns the length of the sequence stored in the SequenceMatcher.
func smSeqLen(sm *object.SequenceMatcher) (int, int) {
	if sm.SeqA != nil {
		return len(sm.SeqA), len(sm.SeqB)
	}
	return len([]rune(sm.A)), len([]rune(sm.B))
}

// smEqual reports whether element i of seqA equals element j of seqB.
func smEqual(sm *object.SequenceMatcher, ai, bj int) bool {
	if sm.SeqA != nil {
		if ai >= len(sm.SeqA) || bj >= len(sm.SeqB) {
			return false
		}
		eq, _ := objectEqual(sm.SeqA[ai], sm.SeqB[bj])
		return eq
	}
	ar, br := []rune(sm.A), []rune(sm.B)
	if ai >= len(ar) || bj >= len(br) {
		return false
	}
	return ar[ai] == br[bj]
}

// objectEqual checks equality between two Objects.
func objectEqual(a, b object.Object) (bool, error) {
	switch va := a.(type) {
	case *object.Str:
		if vb, ok := b.(*object.Str); ok {
			return va.V == vb.V, nil
		}
	case *object.Int:
		if vb, ok := b.(*object.Int); ok {
			return va.V.Cmp(&vb.V) == 0, nil
		}
	case *object.Bool:
		if vb, ok := b.(*object.Bool); ok {
			return va.V == vb.V, nil
		}
	}
	return object.Repr(a) == object.Repr(b), nil
}

// smFindLongestMatch finds the longest matching block in sm.SeqA[alo:ahi]
// and sm.SeqB[blo:bhi], returning (bestI, bestJ, bestSize).
func smFindLongestMatch(sm *object.SequenceMatcher, alo, ahi, blo, bhi int) (int, int, int) {
	bestI, bestJ, bestK := alo, blo, 0
	// j2len[j] = length of longest match ending at sm.SeqB[j]
	j2len := make(map[int]int)
	for i := alo; i < ahi; i++ {
		newj2len := make(map[int]int)
		for j := blo; j < bhi; j++ {
			if smEqual(sm, i, j) {
				k := j2len[j-1] + 1
				newj2len[j] = k
				if k > bestK {
					bestI, bestJ, bestK = i-k+1, j-k+1, k
				}
			}
		}
		j2len = newj2len
	}
	return bestI, bestJ, bestK
}

// smGetMatchingBlocks returns all matching blocks as (a, b, size) triples.
func smGetMatchingBlocks(sm *object.SequenceMatcher) [][3]int {
	la, lb := smSeqLen(sm)
	var blocks [][3]int
	var recurse func(alo, ahi, blo, bhi int)
	recurse = func(alo, ahi, blo, bhi int) {
		i, j, k := smFindLongestMatch(sm, alo, ahi, blo, bhi)
		if k > 0 {
			recurse(alo, i, blo, j)
			blocks = append(blocks, [3]int{i, j, k})
			recurse(i+k, ahi, j+k, bhi)
		}
	}
	recurse(0, la, 0, lb)
	// Sentinel
	blocks = append(blocks, [3]int{la, lb, 0})
	return blocks
}

// smGetOpcodes derives edit opcodes from matching blocks.
func smGetOpcodes(sm *object.SequenceMatcher) [][5]interface{} {
	blocks := smGetMatchingBlocks(sm)
	var opcodes [][5]interface{}
	i, j := 0, 0
	for _, b := range blocks {
		ai, bj, size := b[0], b[1], b[2]
		var tag string
		if i < ai && j < bj {
			tag = "replace"
		} else if i < ai {
			tag = "delete"
		} else if j < bj {
			tag = "insert"
		}
		if tag != "" {
			opcodes = append(opcodes, [5]interface{}{tag, i, ai, j, bj})
		}
		if size > 0 {
			opcodes = append(opcodes, [5]interface{}{"equal", ai, ai + size, bj, bj + size})
		}
		i, j = ai+size, bj+size
	}
	return opcodes
}

// smRatio computes the Ratcliff/Obershelp ratio for a SequenceMatcher.
func smRatio(sm *object.SequenceMatcher) float64 {
	if sm.SeqA != nil {
		la, lb := len(sm.SeqA), len(sm.SeqB)
		if la+lb == 0 {
			return 1.0
		}
		blocks := smGetMatchingBlocks(sm)
		matches := 0
		for _, b := range blocks {
			matches += b[2]
		}
		return 2.0 * float64(matches) / float64(la+lb)
	}
	return ratcliffObershelp(sm.A, sm.B)
}

// makeMatchTuple constructs a Match-like object with .a, .b, .size attrs and
// repr "Match(a=N, b=N, size=N)".
func makeMatchTuple(interp *Interp, a, b, size int) *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	inst.Dict.SetStr("a", object.NewInt(int64(a)))
	inst.Dict.SetStr("b", object.NewInt(int64(b)))
	inst.Dict.SetStr("size", object.NewInt(int64(size)))
	// Give it a synthetic class named "Match" so Repr shows the right name.
	cls := &object.Class{Name: "Match", Dict: object.NewDict()}
	reprStr := fmt.Sprintf("Match(a=%d, b=%d, size=%d)", a, b, size)
	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: reprStr}, nil
	}})
	inst.Class = cls
	return inst
}

// sequenceMatcherAttr dispatches attribute access on *object.SequenceMatcher.
func sequenceMatcherAttr(i *Interp, sm *object.SequenceMatcher, name string) (object.Object, bool) {
	switch name {
	case "a":
		return &object.Str{V: sm.A}, true
	case "b":
		return &object.Str{V: sm.B}, true
	case "ratio":
		return &object.BuiltinFunc{Name: "ratio", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Float{V: smRatio(sm)}, nil
		}}, true
	case "quick_ratio":
		return &object.BuiltinFunc{Name: "quick_ratio", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if sm.SeqA != nil {
				return &object.Float{V: smRatio(sm)}, nil
			}
			return &object.Float{V: quickRatio(sm.A, sm.B)}, nil
		}}, true
	case "real_quick_ratio":
		return &object.BuiltinFunc{Name: "real_quick_ratio", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			la, lb := smSeqLen(sm)
			if la+lb == 0 {
				return &object.Float{V: 1.0}, nil
			}
			m := la
			if lb < m {
				m = lb
			}
			return &object.Float{V: 2.0 * float64(m) / float64(la+lb)}, nil
		}}, true
	case "set_seqs":
		return &object.BuiltinFunc{Name: "set_seqs", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 { smSetSeq(sm, 1, a[0]) }
			if len(a) >= 2 { smSetSeq(sm, 2, a[1]) }
			return object.None, nil
		}}, true
	case "set_seq1":
		return &object.BuiltinFunc{Name: "set_seq1", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 { smSetSeq(sm, 1, a[0]) }
			return object.None, nil
		}}, true
	case "set_seq2":
		return &object.BuiltinFunc{Name: "set_seq2", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 { smSetSeq(sm, 2, a[0]) }
			return object.None, nil
		}}, true
	case "find_longest_match":
		return &object.BuiltinFunc{Name: "find_longest_match", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
			la, lb := smSeqLen(sm)
			alo, ahi, blo, bhi := 0, la, 0, lb
			if len(args) >= 1 { if v, ok := toInt64(args[0]); ok { alo = int(v) } }
			if len(args) >= 2 { if v, ok := toInt64(args[1]); ok { ahi = int(v) } }
			if len(args) >= 3 { if v, ok := toInt64(args[2]); ok { blo = int(v) } }
			if len(args) >= 4 { if v, ok := toInt64(args[3]); ok { bhi = int(v) } }
			if kw != nil {
				if v, ok := kw.GetStr("alo"); ok { if n, ok2 := toInt64(v); ok2 { alo = int(n) } }
				if v, ok := kw.GetStr("ahi"); ok { if n, ok2 := toInt64(v); ok2 { ahi = int(n) } }
				if v, ok := kw.GetStr("blo"); ok { if n, ok2 := toInt64(v); ok2 { blo = int(n) } }
				if v, ok := kw.GetStr("bhi"); ok { if n, ok2 := toInt64(v); ok2 { bhi = int(n) } }
			}
			a, b, size := smFindLongestMatch(sm, alo, ahi, blo, bhi)
			return makeMatchTuple(i, a, b, size), nil
		}}, true
	case "get_matching_blocks":
		return &object.BuiltinFunc{Name: "get_matching_blocks", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			blocks := smGetMatchingBlocks(sm)
			out := make([]object.Object, len(blocks))
			for k, b := range blocks {
				out[k] = makeMatchTuple(i, b[0], b[1], b[2])
			}
			return &object.List{V: out}, nil
		}}, true
	case "get_opcodes":
		return &object.BuiltinFunc{Name: "get_opcodes", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ops := smGetOpcodes(sm)
			out := make([]object.Object, len(ops))
			for k, op := range ops {
				tup := []object.Object{
					&object.Str{V: op[0].(string)},
					object.NewInt(int64(op[1].(int))),
					object.NewInt(int64(op[2].(int))),
					object.NewInt(int64(op[3].(int))),
					object.NewInt(int64(op[4].(int))),
				}
				out[k] = &object.Tuple{V: tup}
			}
			return &object.List{V: out}, nil
		}}, true
	case "get_grouped_opcodes":
		return &object.BuiltinFunc{Name: "get_grouped_opcodes", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
			n := 3
			if len(args) >= 1 { if v, ok := toInt64(args[0]); ok { n = int(v) } }
			if kw != nil { if v, ok := kw.GetStr("n"); ok { if iv, ok2 := toInt64(v); ok2 { n = int(iv) } } }
			ops := smGetOpcodes(sm)
			groups := groupOpcodes(ops, n)
			out := make([]object.Object, len(groups))
			for gi, g := range groups {
				elems := make([]object.Object, len(g))
				for k, op := range g {
					tup := []object.Object{
						&object.Str{V: op[0].(string)},
						object.NewInt(int64(op[1].(int))),
						object.NewInt(int64(op[2].(int))),
						object.NewInt(int64(op[3].(int))),
						object.NewInt(int64(op[4].(int))),
					}
					elems[k] = &object.Tuple{V: tup}
				}
				out[gi] = &object.List{V: elems}
			}
			return &object.List{V: out}, nil
		}}, true
	}
	return nil, false
}

// smSetSeq updates seq 1 (A) or 2 (B) from an Object.
func smSetSeq(sm *object.SequenceMatcher, which int, v object.Object) {
	switch val := v.(type) {
	case *object.Str:
		if which == 1 { sm.A = val.V; sm.SeqA = nil } else { sm.B = val.V; sm.SeqB = nil }
	case *object.List:
		if which == 1 { sm.SeqA = val.V; sm.A = "" } else { sm.SeqB = val.V; sm.B = "" }
	case *object.Tuple:
		if which == 1 { sm.SeqA = val.V; sm.A = "" } else { sm.SeqB = val.V; sm.B = "" }
	}
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

// groupOpcodes groups opcodes into context-bounded hunks (like CPython's
// SequenceMatcher.get_grouped_opcodes).
func groupOpcodes(ops [][5]interface{}, n int) [][][5]interface{} {
	if len(ops) == 0 {
		return nil
	}
	// Trim leading/trailing equal blocks to n lines.
	trimmed := make([][5]interface{}, len(ops))
	copy(trimmed, ops)
	if len(trimmed) > 0 && trimmed[0][0] == "equal" {
		op := trimmed[0]
		i1, i2, _, j2 := op[1].(int), op[2].(int), op[3].(int), op[4].(int)
		if i2-i1 > n {
			trimmed[0] = [5]interface{}{"equal", i2 - n, i2, j2 - n, j2}
		}
	}
	if len(trimmed) > 0 && trimmed[len(trimmed)-1][0] == "equal" {
		op := trimmed[len(trimmed)-1]
		i1, i2, j1, _ := op[1].(int), op[2].(int), op[3].(int), op[4].(int)
		if i2-i1 > n {
			trimmed[len(trimmed)-1] = [5]interface{}{"equal", i1, i1 + n, j1, j1 + n}
		}
	}
	var groups [][][5]interface{}
	var group [][5]interface{}
	for _, op := range trimmed {
		if op[0] == "equal" && op[2].(int)-op[1].(int) > 2*n {
			// Split large equal block: flush current group, start new one.
			i1, i2, j1, j2 := op[1].(int), op[2].(int), op[3].(int), op[4].(int)
			group = append(group, [5]interface{}{"equal", i1, i1 + n, j1, j1 + n})
			groups = append(groups, group)
			group = [][5]interface{}{{op[0], i2 - n, i2, j2 - n, j2}}
		} else {
			group = append(group, op)
		}
	}
	if len(group) > 0 {
		// Only flush if there's at least one non-equal op in the group.
		hasChange := false
		for _, op := range group {
			if op[0] != "equal" {
				hasChange = true
				break
			}
		}
		if hasChange {
			groups = append(groups, group)
		}
	}
	return groups
}

// unifiedDiff produces CPython-compatible unified diff output with n context lines.
func unifiedDiff(a, b []string, fromfile, tofile, fromdate, todate string, n int, lineterm string) []string {
	sm := &object.SequenceMatcher{SeqA: strSliceToObjs(a), SeqB: strSliceToObjs(b)}
	ops := smGetOpcodes(sm)
	groups := groupOpcodes(ops, n)
	if len(groups) == 0 {
		return nil
	}
	var lines []string
	fromhdr := fromfile
	if fromdate != "" { fromhdr += "\t" + fromdate }
	tohdr := tofile
	if todate != "" { tohdr += "\t" + todate }
	lines = append(lines, "--- "+fromhdr+lineterm)
	lines = append(lines, "+++ "+tohdr+lineterm)
	for _, g := range groups {
		first, last := g[0], g[len(g)-1]
		i1 := first[1].(int)
		i2 := last[2].(int)
		j1 := first[3].(int)
		j2 := last[4].(int)
		lines = append(lines, fmt.Sprintf("@@ -%d,%d +%d,%d @@%s", i1+1, i2-i1, j1+1, j2-j1, lineterm))
		for _, op := range g {
			tag := op[0].(string)
			oi1, oi2, oj1, oj2 := op[1].(int), op[2].(int), op[3].(int), op[4].(int)
			switch tag {
			case "equal":
				for _, l := range a[oi1:oi2] { lines = append(lines, " "+l) }
			case "replace":
				for _, l := range a[oi1:oi2] { lines = append(lines, "-"+l) }
				for _, l := range b[oj1:oj2] { lines = append(lines, "+"+l) }
			case "delete":
				for _, l := range a[oi1:oi2] { lines = append(lines, "-"+l) }
			case "insert":
				for _, l := range b[oj1:oj2] { lines = append(lines, "+"+l) }
			}
		}
	}
	return lines
}

// contextDiff produces CPython-compatible context diff output.
func contextDiff(a, b []string, fromfile, tofile, fromdate, todate string, n int, lineterm string) []string {
	sm := &object.SequenceMatcher{SeqA: strSliceToObjs(a), SeqB: strSliceToObjs(b)}
	ops := smGetOpcodes(sm)
	groups := groupOpcodes(ops, n)
	if len(groups) == 0 {
		return nil
	}
	var lines []string
	fromhdr := fromfile
	if fromdate != "" { fromhdr += "\t" + fromdate }
	tohdr := tofile
	if todate != "" { tohdr += "\t" + todate }
	lines = append(lines, "*** "+fromhdr+lineterm)
	lines = append(lines, "--- "+tohdr+lineterm)
	for _, g := range groups {
		first, last := g[0], g[len(g)-1]
		i1 := first[1].(int) + 1
		i2 := last[2].(int)
		j1 := first[3].(int) + 1
		j2 := last[4].(int)
		lines = append(lines, "***************"+lineterm)
		lines = append(lines, fmt.Sprintf("*** %d,%d ****%s", i1, i2, lineterm))
		// from side
		hasChange := false
		for _, op := range g {
			if op[0].(string) != "insert" { hasChange = true; break }
		}
		if hasChange {
			for _, op := range g {
				tag := op[0].(string)
				oi1, oi2 := op[1].(int), op[2].(int)
				switch tag {
				case "equal":
					for _, l := range a[oi1:oi2] { lines = append(lines, "  "+l) }
				case "replace", "delete":
					for _, l := range a[oi1:oi2] { lines = append(lines, "! "+l) }
				}
			}
		}
		lines = append(lines, fmt.Sprintf("--- %d,%d ----%s", j1, j2, lineterm))
		// to side
		hasChange2 := false
		for _, op := range g {
			if op[0].(string) != "delete" { hasChange2 = true; break }
		}
		if hasChange2 {
			for _, op := range g {
				tag := op[0].(string)
				oj1, oj2 := op[3].(int), op[4].(int)
				switch tag {
				case "equal":
					for _, l := range b[oj1:oj2] { lines = append(lines, "  "+l) }
				case "replace", "insert":
					for _, l := range b[oj1:oj2] { lines = append(lines, "! "+l) }
				}
			}
		}
	}
	return lines
}

// strSliceToObjs converts a []string to []object.Object for use with SequenceMatcher.
func strSliceToObjs(ss []string) []object.Object {
	out := make([]object.Object, len(ss))
	for k, s := range ss {
		out[k] = &object.Str{V: s}
	}
	return out
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
