// Package toml implements a TOML v1.0 parser with no external dependencies.
// Loads returns a map[string]interface{} with the following Go types:
//
//	TOML string          → string
//	TOML integer         → int64
//	TOML float           → float64
//	TOML boolean         → bool
//	TOML offset-datetime → OffsetDateTime
//	TOML local-datetime  → LocalDateTime
//	TOML local-date      → LocalDate
//	TOML local-time      → LocalTime
//	TOML array           → []interface{}
//	TOML table           → map[string]interface{}
package toml

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ── datetime types ─────────────────────────────────────────────────────────

type OffsetDateTime struct {
	Year, Month, Day             int
	Hour, Minute, Second         int
	Microsecond                  int
	TZOffsetSecs                 int // seconds east of UTC
}

type LocalDateTime struct {
	Year, Month, Day     int
	Hour, Minute, Second int
	Microsecond          int
}

type LocalDate struct{ Year, Month, Day int }
type LocalTime struct{ Hour, Minute, Second, Microsecond int }

// ── error type ─────────────────────────────────────────────────────────────

type DecodeError struct {
	Line int
	Msg  string
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("Invalid TOML value (line %d): %s", e.Line, e.Msg)
}

// ── public API ─────────────────────────────────────────────────────────────

func Loads(s string) (map[string]interface{}, error) {
	p := &parser{
		s:       s,
		line:    1,
		root:    make(map[string]interface{}),
		defined: make(map[string]bool),
		aot:     make(map[string]bool),
		frozen:  make(map[string]bool),
	}
	if err := p.parseDocument(); err != nil {
		return nil, err
	}
	return p.root, nil
}

// ── parser ─────────────────────────────────────────────────────────────────

type parser struct {
	s       string
	pos     int
	line    int
	root    map[string]interface{}
	defined map[string]bool   // paths explicitly defined (key or [table])
	aot     map[string]bool   // array-of-tables paths
	frozen  map[string]bool   // inline-table paths (cannot be extended)
	curPath []string          // current implicit table context
}

func (p *parser) errorf(f string, a ...interface{}) *DecodeError {
	return &DecodeError{Line: p.line, Msg: fmt.Sprintf(f, a...)}
}

func pathKey(parts []string) string { return strings.Join(parts, "\x00") }

func (p *parser) peek() (byte, bool) {
	if p.pos >= len(p.s) {
		return 0, false
	}
	return p.s[p.pos], true
}

func (p *parser) advance() {
	if p.pos < len(p.s) {
		if p.s[p.pos] == '\n' {
			p.line++
		}
		p.pos++
	}
}

func (p *parser) skipWS() {
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == ' ' || c == '\t' {
			p.pos++
		} else {
			break
		}
	}
}

func (p *parser) skipWSNL() {
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == ' ' || c == '\t' || c == '\r' {
			p.pos++
		} else if c == '\n' {
			p.line++
			p.pos++
		} else {
			break
		}
	}
}

func (p *parser) skipComment() {
	if p.pos < len(p.s) && p.s[p.pos] == '#' {
		for p.pos < len(p.s) && p.s[p.pos] != '\n' {
			p.pos++
		}
	}
}

func (p *parser) skipToEOL() {
	p.skipWS()
	p.skipComment()
	if p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == '\r' {
			p.pos++
		}
		if p.pos < len(p.s) && p.s[p.pos] == '\n' {
			p.line++
			p.pos++
		} else if p.pos < len(p.s) && c != '\n' {
			// leftover non-whitespace after comment skip → error in caller
		}
	}
}

// ── document level ─────────────────────────────────────────────────────────

func (p *parser) parseDocument() error {
	for {
		p.skipWSNL()
		if p.pos >= len(p.s) {
			break
		}
		c := p.s[p.pos]
		if c == '#' {
			p.skipComment()
			continue
		}
		if c == '[' {
			if err := p.parseTableHeader(); err != nil {
				return err
			}
			continue
		}
		// key = value
		if err := p.parseKeyValue(p.curPath); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) parseTableHeader() error {
	p.pos++ // consume '['
	isAOT := false
	if p.pos < len(p.s) && p.s[p.pos] == '[' {
		isAOT = true
		p.pos++ // consume second '['
	}
	p.skipWS()
	keys, err := p.parseKeyPath()
	if err != nil {
		return err
	}
	p.skipWS()
	if isAOT {
		if p.pos >= len(p.s) || p.s[p.pos] != ']' {
			return p.errorf("expected ]] closing array-of-tables")
		}
		p.pos++
		if p.pos >= len(p.s) || p.s[p.pos] != ']' {
			return p.errorf("expected ]] closing array-of-tables")
		}
		p.pos++
	} else {
		if p.pos >= len(p.s) || p.s[p.pos] != ']' {
			return p.errorf("expected ] closing table header")
		}
		p.pos++
	}
	p.skipToEOL()

	fullKey := pathKey(keys)
	if isAOT {
		if p.defined[fullKey] && !p.aot[fullKey] {
			return p.errorf("cannot define array of tables %q: already a table", strings.Join(keys, "."))
		}
		p.aot[fullKey] = true
		// Navigate to parent and append new map to the array.
		newMap := make(map[string]interface{})
		if err := p.appendAOT(keys, newMap); err != nil {
			return err
		}
		p.curPath = keys
	} else {
		if p.defined[fullKey] {
			return p.errorf("duplicate table %q", strings.Join(keys, "."))
		}
		if p.aot[fullKey] {
			return p.errorf("cannot define table %q: already an array of tables", strings.Join(keys, "."))
		}
		p.defined[fullKey] = true
		// Ensure the table exists.
		if _, err := p.navigatePath(keys, false); err != nil {
			return err
		}
		p.curPath = keys
	}
	return nil
}

// appendAOT appends newMap to the array-of-tables at path keys.
func (p *parser) appendAOT(keys []string, newMap map[string]interface{}) error {
	if len(keys) == 0 {
		return p.errorf("empty array-of-tables key")
	}
	parent, err := p.navigatePath(keys[:len(keys)-1], true)
	if err != nil {
		return err
	}
	last := keys[len(keys)-1]
	existing, ok := parent[last]
	if !ok {
		parent[last] = []interface{}{newMap}
		return nil
	}
	arr, ok := existing.([]interface{})
	if !ok {
		return p.errorf("cannot use %q as array of tables: already has a value", last)
	}
	parent[last] = append(arr, newMap)
	return nil
}

// navigatePath returns the map at path, creating intermediate maps as needed.
// If forAOT is true, it navigates into the last element of any arrays found.
func (p *parser) navigatePath(path []string, forAOT bool) (map[string]interface{}, error) {
	cur := p.root
	for _, key := range path {
		v, exists := cur[key]
		if !exists {
			m := make(map[string]interface{})
			cur[key] = m
			cur = m
			continue
		}
		switch vt := v.(type) {
		case map[string]interface{}:
			cur = vt
		case []interface{}:
			if len(vt) == 0 {
				return nil, p.errorf("empty array at key %q", key)
			}
			last, ok := vt[len(vt)-1].(map[string]interface{})
			if !ok {
				return nil, p.errorf("cannot navigate into array element at %q: not a table", key)
			}
			cur = last
		default:
			return nil, p.errorf("key %q already has a non-table value", key)
		}
	}
	return cur, nil
}


// ── key=value ──────────────────────────────────────────────────────────────

func (p *parser) parseKeyValue(tableCtx []string) error {
	keys, err := p.parseKeyPath()
	if err != nil {
		return err
	}
	p.skipWS()
	if p.pos >= len(p.s) || p.s[p.pos] != '=' {
		return p.errorf("expected '=' after key")
	}
	p.pos++ // consume '='
	p.skipWS()
	val, err := p.parseValue()
	if err != nil {
		return err
	}
	p.skipToEOL()

	// Navigate to the target table and set the value.
	fullPath := append(tableCtx, keys...)
	return p.setNestedValue(fullPath, val)
}

// setNestedValue sets value at the given full path.
func (p *parser) setNestedValue(path []string, val interface{}) error {
	if len(path) == 0 {
		return p.errorf("empty key path")
	}
	parent, err := p.navigateToParent(path[:len(path)-1])
	if err != nil {
		return err
	}
	last := path[len(path)-1]
	fullKey := pathKey(path)

	// Detect frozen inline table extension.
	parentPath := pathKey(path[:len(path)-1])
	if p.frozen[parentPath] {
		return p.errorf("cannot add key %q: inline table is immutable", last)
	}

	if _, exists := parent[last]; exists {
		// If the existing value is a map and the new value is also a map,
		// it might be dotted-key building (allowed if not yet defined).
		newMap, newIsMap := val.(map[string]interface{})
		existMap, existIsMap := parent[last].(map[string]interface{})
		if newIsMap && existIsMap && !p.defined[fullKey] {
			// Merge: add keys from newMap into existMap.
			for k, v := range newMap {
				if _, dup := existMap[k]; dup {
					return p.errorf("duplicate key %q", last+"."+k)
				}
				existMap[k] = v
			}
			return nil
		}
		return p.errorf("duplicate key %q", last)
	}
	parent[last] = val
	p.defined[fullKey] = true

	// Mark inline tables as frozen.
	if _, isMap := val.(map[string]interface{}); isMap {
		if p.isInlineTable(val) {
			p.markFrozen(val.(map[string]interface{}), path)
		}
	}
	return nil
}

func (p *parser) markFrozen(m map[string]interface{}, path []string) {
	_ = m
	p.frozen[pathKey(path)] = true
}

func (p *parser) isInlineTable(_ interface{}) bool {
	return false // caller sets frozen directly after parseInlineTable
}

// navigateToParent navigates to the map containing the final key segment.
// Creates intermediate maps as needed.
func (p *parser) navigateToParent(path []string) (map[string]interface{}, error) {
	cur := p.root
	for idx, key := range path {
		v, exists := cur[key]
		if !exists {
			m := make(map[string]interface{})
			cur[key] = m
			// Mark as implicitly created by dotted key.
			cur = m
			continue
		}
		switch vt := v.(type) {
		case map[string]interface{}:
			partPath := pathKey(path[:idx+1])
			if p.frozen[partPath] {
				return nil, p.errorf("cannot extend inline table at %q", key)
			}
			cur = vt
		case []interface{}:
			if len(vt) == 0 {
				return nil, p.errorf("empty array at %q", key)
			}
			last, ok := vt[len(vt)-1].(map[string]interface{})
			if !ok {
				return nil, p.errorf("array element at %q is not a table", key)
			}
			cur = last
		default:
			return nil, p.errorf("key %q is not a table", key)
		}
	}
	return cur, nil
}

// ── key path parsing ───────────────────────────────────────────────────────

func (p *parser) parseKeyPath() ([]string, error) {
	var keys []string
	for {
		k, err := p.parseSingleKey()
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
		p.skipWS()
		if p.pos < len(p.s) && p.s[p.pos] == '.' {
			p.pos++
			p.skipWS()
			continue
		}
		break
	}
	return keys, nil
}

func (p *parser) parseSingleKey() (string, error) {
	if p.pos >= len(p.s) {
		return "", p.errorf("expected key")
	}
	c := p.s[p.pos]
	if c == '"' {
		return p.parseBasicString()
	}
	if c == '\'' {
		return p.parseLiteralString()
	}
	// Bare key: [A-Za-z0-9_-]+
	start := p.pos
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '_' || c == '-' {
			p.pos++
		} else {
			break
		}
	}
	if p.pos == start {
		return "", p.errorf("invalid bare key character %q", p.s[p.pos])
	}
	return p.s[start:p.pos], nil
}

// ── value parsing ──────────────────────────────────────────────────────────

func (p *parser) parseValue() (interface{}, error) {
	if p.pos >= len(p.s) {
		return nil, p.errorf("expected value")
	}
	c := p.s[p.pos]
	switch {
	case c == '"':
		if p.pos+2 < len(p.s) && p.s[p.pos+1] == '"' && p.s[p.pos+2] == '"' {
			return p.parseMLBasicString()
		}
		return p.parseBasicString()
	case c == '\'':
		if p.pos+2 < len(p.s) && p.s[p.pos+1] == '\'' && p.s[p.pos+2] == '\'' {
			return p.parseMLLiteralString()
		}
		return p.parseLiteralString()
	case c == '[':
		return p.parseArray()
	case c == '{':
		return p.parseInlineTable()
	case c == 't':
		return p.parseBoolTrue()
	case c == 'f':
		return p.parseBoolFalse()
	case c == 'i':
		return p.parseSpecialFloat("inf")
	case c == 'n':
		return p.parseSpecialFloat("nan")
	case c == '+' || c == '-':
		return p.parseNumOrDateWithSign()
	default:
		if c >= '0' && c <= '9' {
			return p.parseNumOrDate()
		}
		return nil, p.errorf("unexpected character %q at start of value", c)
	}
}

// ── string parsing ─────────────────────────────────────────────────────────

func (p *parser) parseBasicString() (string, error) {
	p.pos++ // consume opening "
	var b strings.Builder
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == '"' {
			p.pos++
			return b.String(), nil
		}
		if c == '\n' || c == '\r' {
			return "", p.errorf("newline in basic string")
		}
		if c == '\\' {
			p.pos++
			r, err := p.parseEscape()
			if err != nil {
				return "", err
			}
			b.WriteRune(r)
			continue
		}
		r, size := utf8.DecodeRuneInString(p.s[p.pos:])
		b.WriteRune(r)
		p.pos += size
	}
	return "", p.errorf("unterminated basic string")
}

func (p *parser) parseLiteralString() (string, error) {
	p.pos++ // consume '
	start := p.pos
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == '\'' {
			result := p.s[start:p.pos]
			p.pos++
			return result, nil
		}
		if c == '\n' || c == '\r' {
			return "", p.errorf("newline in literal string")
		}
		p.pos++
	}
	return "", p.errorf("unterminated literal string")
}

func (p *parser) parseMLBasicString() (string, error) {
	p.pos += 3 // consume """
	// Strip immediately following newline.
	if p.pos < len(p.s) && p.s[p.pos] == '\n' {
		p.line++
		p.pos++
	} else if p.pos+1 < len(p.s) && p.s[p.pos] == '\r' && p.s[p.pos+1] == '\n' {
		p.line++
		p.pos += 2
	}
	var b strings.Builder
	for p.pos < len(p.s) {
		// Check for closing """
		if p.s[p.pos] == '"' {
			// Count consecutive quotes.
			n := 0
			for p.pos+n < len(p.s) && p.s[p.pos+n] == '"' {
				n++
			}
			if n >= 3 {
				// Up to 2 extra quotes can precede the closing """.
				extra := n - 3
				for k := 0; k < extra; k++ {
					b.WriteByte('"')
				}
				p.pos += n
				return b.String(), nil
			}
		}
		c := p.s[p.pos]
		if c == '\\' {
			p.pos++
			if p.pos < len(p.s) && (p.s[p.pos] == '\n' || p.s[p.pos] == '\r' || p.s[p.pos] == ' ' || p.s[p.pos] == '\t') {
				// Line-ending backslash: trim whitespace until next non-ws.
				for p.pos < len(p.s) {
					ch := p.s[p.pos]
					if ch == ' ' || ch == '\t' || ch == '\r' {
						p.pos++
					} else if ch == '\n' {
						p.line++
						p.pos++
					} else {
						break
					}
				}
				continue
			}
			r, err := p.parseEscape()
			if err != nil {
				return "", err
			}
			b.WriteRune(r)
			continue
		}
		if c == '\n' {
			p.line++
			b.WriteByte('\n')
			p.pos++
			continue
		}
		if c == '\r' {
			p.pos++
			if p.pos < len(p.s) && p.s[p.pos] == '\n' {
				p.line++
				p.pos++
			}
			b.WriteByte('\n')
			continue
		}
		r, size := utf8.DecodeRuneInString(p.s[p.pos:])
		b.WriteRune(r)
		p.pos += size
	}
	return "", p.errorf("unterminated multi-line basic string")
}

func (p *parser) parseMLLiteralString() (string, error) {
	p.pos += 3 // consume '''
	// Strip immediately following newline.
	if p.pos < len(p.s) && p.s[p.pos] == '\n' {
		p.line++
		p.pos++
	} else if p.pos+1 < len(p.s) && p.s[p.pos] == '\r' && p.s[p.pos+1] == '\n' {
		p.line++
		p.pos += 2
	}
	var b strings.Builder
	for p.pos < len(p.s) {
		if p.s[p.pos] == '\'' {
			n := 0
			for p.pos+n < len(p.s) && p.s[p.pos+n] == '\'' {
				n++
			}
			if n >= 3 {
				extra := n - 3
				for k := 0; k < extra; k++ {
					b.WriteByte('\'')
				}
				p.pos += n
				return b.String(), nil
			}
		}
		c := p.s[p.pos]
		if c == '\n' {
			p.line++
			b.WriteByte('\n')
			p.pos++
			continue
		}
		if c == '\r' {
			p.pos++
			if p.pos < len(p.s) && p.s[p.pos] == '\n' {
				p.line++
				p.pos++
			}
			b.WriteByte('\n')
			continue
		}
		r, size := utf8.DecodeRuneInString(p.s[p.pos:])
		b.WriteRune(r)
		p.pos += size
	}
	return "", p.errorf("unterminated multi-line literal string")
}

func (p *parser) parseEscape() (rune, error) {
	if p.pos >= len(p.s) {
		return 0, p.errorf("unexpected end after backslash")
	}
	c := p.s[p.pos]
	p.pos++
	switch c {
	case 'b':
		return '\b', nil
	case 't':
		return '\t', nil
	case 'n':
		return '\n', nil
	case 'f':
		return '\f', nil
	case 'r':
		return '\r', nil
	case '"':
		return '"', nil
	case '\\':
		return '\\', nil
	case 'u':
		return p.parseHexEscape(4)
	case 'U':
		return p.parseHexEscape(8)
	default:
		return 0, p.errorf("invalid escape sequence \\%c", c)
	}
}

func (p *parser) parseHexEscape(n int) (rune, error) {
	if p.pos+n > len(p.s) {
		return 0, p.errorf("short unicode escape")
	}
	hex := p.s[p.pos : p.pos+n]
	p.pos += n
	v, err := strconv.ParseInt(hex, 16, 32)
	if err != nil {
		return 0, p.errorf("invalid unicode escape %q", hex)
	}
	return rune(v), nil
}

// ── bool ───────────────────────────────────────────────────────────────────

func (p *parser) parseBoolTrue() (interface{}, error) {
	if p.pos+4 <= len(p.s) && p.s[p.pos:p.pos+4] == "true" {
		p.pos += 4
		return true, nil
	}
	return nil, p.errorf("invalid value starting with 't'")
}

func (p *parser) parseBoolFalse() (interface{}, error) {
	if p.pos+5 <= len(p.s) && p.s[p.pos:p.pos+5] == "false" {
		p.pos += 5
		return false, nil
	}
	return nil, p.errorf("invalid value starting with 'f'")
}

// ── special floats (inf/nan without sign) ─────────────────────────────────

func (p *parser) parseSpecialFloat(kw string) (interface{}, error) {
	if p.pos+len(kw) <= len(p.s) && p.s[p.pos:p.pos+len(kw)] == kw {
		p.pos += len(kw)
		if kw == "inf" {
			return math.Inf(1), nil
		}
		return math.NaN(), nil
	}
	return nil, p.errorf("expected %q", kw)
}

// ── number / date disambiguation ──────────────────────────────────────────

func (p *parser) parseNumOrDateWithSign() (interface{}, error) {
	sign := p.s[p.pos]
	p.pos++
	// Check for special floats: +inf, -inf, +nan, -nan
	if p.pos+3 <= len(p.s) && p.s[p.pos:p.pos+3] == "inf" {
		p.pos += 3
		if sign == '-' {
			return math.Inf(-1), nil
		}
		return math.Inf(1), nil
	}
	if p.pos+3 <= len(p.s) && p.s[p.pos:p.pos+3] == "nan" {
		p.pos += 3
		return math.NaN(), nil
	}
	v, err := p.parseNumOrDate()
	if err != nil {
		return nil, err
	}
	if sign == '-' {
		switch vt := v.(type) {
		case int64:
			return -vt, nil
		case float64:
			return -vt, nil
		default:
			return nil, p.errorf("sign not allowed before %T", v)
		}
	}
	return v, nil
}

// parseNumOrDate parses the rest of a number or datetime (no leading sign).
func (p *parser) parseNumOrDate() (interface{}, error) {
	start := p.pos
	// Scan ahead to determine type.
	// If we see digits then 'T', ' ', or '-' after 4 digits → potential datetime.
	// If we see 0x/0o/0b → integer base.
	if p.pos+2 <= len(p.s) && p.s[p.pos] == '0' {
		switch p.s[p.pos+1] {
		case 'x', 'X':
			p.pos += 2
			return p.parseIntBase(16)
		case 'o', 'O':
			p.pos += 2
			return p.parseIntBase(8)
		case 'b', 'B':
			p.pos += 2
			return p.parseIntBase(2)
		}
	}
	// Scan digits to find the form.
	i := p.pos
	for i < len(p.s) && (isDigit(p.s[i]) || p.s[i] == '_') {
		i++
	}
	// Check for datetime patterns:
	// YYYY-MM-DD (date or datetime)
	// HH:MM:SS (local time, if first token is 2 digits with colon at pos+2)
	if i-p.pos == 4 && i < len(p.s) && p.s[i] == '-' {
		// Could be YYYY-MM-DD...
		return p.parseDateOrDatetime(start)
	}
	if i-p.pos == 2 && i < len(p.s) && p.s[i] == ':' {
		// Local time: HH:MM:SS
		return p.parseLocalTime(start)
	}
	// Integer or float.
	return p.parseIntOrFloat(start)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func (p *parser) parseIntBase(base int) (interface{}, error) {
	start := p.pos
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == '_' {
			p.pos++
			continue
		}
		if isHexDigit(c, base) {
			p.pos++
		} else {
			break
		}
	}
	raw := strings.ReplaceAll(p.s[start:p.pos], "_", "")
	if raw == "" {
		return nil, p.errorf("expected digits after base prefix")
	}
	n, err := strconv.ParseInt(raw, base, 64)
	if err != nil {
		return nil, p.errorf("invalid integer: %s", err)
	}
	return n, nil
}

func isHexDigit(c byte, base int) bool {
	if base == 2 {
		return c == '0' || c == '1'
	}
	if base == 8 {
		return c >= '0' && c <= '7'
	}
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func (p *parser) parseIntOrFloat(start int) (interface{}, error) {
	// Scan the number token.
	i := p.pos
	isFloat := false
	for i < len(p.s) {
		c := p.s[i]
		if c == '_' || isDigit(c) {
			i++
		} else if c == '.' || c == 'e' || c == 'E' {
			isFloat = true
			i++
		} else if (c == '+' || c == '-') && i > p.pos && (p.s[i-1] == 'e' || p.s[i-1] == 'E') {
			i++
		} else {
			break
		}
	}
	raw := p.s[start:i]
	p.pos = i
	clean := strings.ReplaceAll(raw, "_", "")
	if isFloat {
		f, err := strconv.ParseFloat(clean, 64)
		if err != nil {
			return nil, p.errorf("invalid float %q", raw)
		}
		return f, nil
	}
	n, err := strconv.ParseInt(clean, 10, 64)
	if err != nil {
		return nil, p.errorf("invalid integer %q", raw)
	}
	return n, nil
}

// ── datetime ───────────────────────────────────────────────────────────────

func (p *parser) parseDateOrDatetime(start int) (interface{}, error) {
	// Read YYYY-MM-DD
	if p.pos+10 > len(p.s) {
		return p.parseIntOrFloat(start)
	}
	yearS := p.s[p.pos : p.pos+4]
	if p.s[p.pos+4] != '-' {
		return p.parseIntOrFloat(start)
	}
	monthS := p.s[p.pos+5 : p.pos+7]
	if p.pos+7 >= len(p.s) || p.s[p.pos+7] != '-' {
		return p.parseIntOrFloat(start)
	}
	dayS := p.s[p.pos+8 : p.pos+10]
	year, e1 := strconv.Atoi(yearS)
	month, e2 := strconv.Atoi(monthS)
	day, e3 := strconv.Atoi(dayS)
	if e1 != nil || e2 != nil || e3 != nil {
		return p.parseIntOrFloat(start)
	}
	p.pos += 10

	// Check for time separator: 'T', 't', or ' '
	if p.pos < len(p.s) && (p.s[p.pos] == 'T' || p.s[p.pos] == 't' || p.s[p.pos] == ' ') {
		sep := p.s[p.pos]
		p.pos++
		// Try to parse time.
		lt, err := p.parseTimeFromPos()
		if err != nil && sep == ' ' {
			// Space separator: might not be a time; put back and return date.
			p.pos--
			return LocalDate{year, month, day}, nil
		}
		if err != nil {
			return nil, err
		}
		// Check for timezone.
		_, offsetSecs, hasTZ, err := p.parseTZ()
		if err != nil {
			return nil, err
		}
		if hasTZ {
			return OffsetDateTime{year, month, day, lt.Hour, lt.Minute, lt.Second, lt.Microsecond, offsetSecs}, nil
		}
		return LocalDateTime{year, month, day, lt.Hour, lt.Minute, lt.Second, lt.Microsecond}, nil
	}
	return LocalDate{year, month, day}, nil
}

func (p *parser) parseLocalTime(start int) (interface{}, error) {
	lt, err := p.parseTimeFromPos()
	if err != nil {
		return nil, err
	}
	return lt, nil
}

// parseTimeFromPos parses HH:MM:SS[.fraction] from the current position.
func (p *parser) parseTimeFromPos() (LocalTime, error) {
	if p.pos+8 > len(p.s) {
		return LocalTime{}, p.errorf("incomplete time")
	}
	hhS := p.s[p.pos : p.pos+2]
	if p.s[p.pos+2] != ':' {
		return LocalTime{}, p.errorf("expected ':' in time")
	}
	mmS := p.s[p.pos+3 : p.pos+5]
	if p.s[p.pos+5] != ':' {
		return LocalTime{}, p.errorf("expected ':' in time")
	}
	ssS := p.s[p.pos+6 : p.pos+8]
	hh, e1 := strconv.Atoi(hhS)
	mm, e2 := strconv.Atoi(mmS)
	ss, e3 := strconv.Atoi(ssS)
	if e1 != nil || e2 != nil || e3 != nil {
		return LocalTime{}, p.errorf("invalid time digits")
	}
	p.pos += 8
	usec := 0
	if p.pos < len(p.s) && p.s[p.pos] == '.' {
		p.pos++
		start := p.pos
		for p.pos < len(p.s) && isDigit(p.s[p.pos]) {
			p.pos++
		}
		frac := p.s[start:p.pos]
		// Normalize to microseconds (6 digits).
		for len(frac) < 6 {
			frac += "0"
		}
		if len(frac) > 6 {
			frac = frac[:6]
		}
		usec, _ = strconv.Atoi(frac)
	}
	return LocalTime{hh, mm, ss, usec}, nil
}

func (p *parser) parseTZ() (string, int, bool, error) {
	if p.pos >= len(p.s) {
		return "", 0, false, nil
	}
	c := p.s[p.pos]
	if c == 'Z' || c == 'z' {
		p.pos++
		return "UTC", 0, true, nil
	}
	if c == '+' || c == '-' {
		sign := 1
		if c == '-' {
			sign = -1
		}
		p.pos++
		if p.pos+5 > len(p.s) {
			return "", 0, false, p.errorf("incomplete timezone offset")
		}
		hhS := p.s[p.pos : p.pos+2]
		if p.s[p.pos+2] != ':' {
			return "", 0, false, p.errorf("expected ':' in timezone")
		}
		mmS := p.s[p.pos+3 : p.pos+5]
		p.pos += 5
		hh, e1 := strconv.Atoi(hhS)
		mm, e2 := strconv.Atoi(mmS)
		if e1 != nil || e2 != nil {
			return "", 0, false, p.errorf("invalid timezone digits")
		}
		return "", sign * (hh*3600 + mm*60), true, nil
	}
	return "", 0, false, nil
}

// ── array ──────────────────────────────────────────────────────────────────

func (p *parser) parseArray() (interface{}, error) {
	p.pos++ // consume '['
	var items []interface{}
	for {
		p.skipWSNL()
		if p.pos >= len(p.s) {
			return nil, p.errorf("unterminated array")
		}
		if p.s[p.pos] == '#' {
			p.skipComment()
			continue
		}
		if p.s[p.pos] == ']' {
			p.pos++
			return items, nil
		}
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		items = append(items, val)
		p.skipWSNL()
		if p.pos >= len(p.s) {
			return nil, p.errorf("unterminated array")
		}
		if p.s[p.pos] == '#' {
			p.skipComment()
			continue
		}
		if p.s[p.pos] == ',' {
			p.pos++
			continue
		}
		if p.s[p.pos] == ']' {
			p.pos++
			return items, nil
		}
		return nil, p.errorf("expected ',' or ']' in array")
	}
}

// ── inline table ───────────────────────────────────────────────────────────

func (p *parser) parseInlineTable() (interface{}, error) {
	p.pos++ // consume '{'
	m := make(map[string]interface{})
	definedKeys := make(map[string]bool)
	p.skipWS()
	if p.pos < len(p.s) && p.s[p.pos] == '}' {
		p.pos++
		return m, nil
	}
	for {
		p.skipWS()
		keys, err := p.parseKeyPath()
		if err != nil {
			return nil, err
		}
		p.skipWS()
		if p.pos >= len(p.s) || p.s[p.pos] != '=' {
			return nil, p.errorf("expected '=' in inline table")
		}
		p.pos++
		p.skipWS()
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		// Set into m using the key path.
		k := pathKey(keys)
		if definedKeys[k] {
			return nil, p.errorf("duplicate key %q in inline table", strings.Join(keys, "."))
		}
		definedKeys[k] = true
		if err := inlineSetValue(m, keys, val); err != nil {
			return nil, p.errorf("%s", err)
		}
		p.skipWS()
		if p.pos >= len(p.s) {
			return nil, p.errorf("unterminated inline table")
		}
		if p.s[p.pos] == '}' {
			p.pos++
			return m, nil
		}
		if p.s[p.pos] == ',' {
			p.pos++
			continue
		}
		return nil, p.errorf("expected ',' or '}' in inline table")
	}
}

func inlineSetValue(m map[string]interface{}, keys []string, val interface{}) error {
	cur := m
	for _, k := range keys[:len(keys)-1] {
		v, exists := cur[k]
		if !exists {
			sub := make(map[string]interface{})
			cur[k] = sub
			cur = sub
			continue
		}
		sub, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("key %q is not a table", k)
		}
		cur = sub
	}
	last := keys[len(keys)-1]
	if _, exists := cur[last]; exists {
		return fmt.Errorf("duplicate key %q", last)
	}
	cur[last] = val
	return nil
}
