package object

// --- configparser types -----------------------------------------------------

// CfgSection holds the options for one INI section in insertion order.
type CfgSection struct {
	Keys   []string
	Values map[string]string
	NoVal  map[string]bool // keys with no value (allow_no_value)
}

func NewCfgSection() *CfgSection {
	return &CfgSection{Values: make(map[string]string), NoVal: make(map[string]bool)}
}

func (s *CfgSection) Set(key, val string) {
	if _, exists := s.Values[key]; !exists {
		s.Keys = append(s.Keys, key)
	}
	s.Values[key] = val
	delete(s.NoVal, key)
}

func (s *CfgSection) SetNoVal(key string) {
	if _, exists := s.Values[key]; !exists {
		if !s.NoVal[key] {
			s.Keys = append(s.Keys, key)
		}
	}
	s.NoVal[key] = true
}

func (s *CfgSection) Del(key string) bool {
	if _, ok := s.Values[key]; ok {
		delete(s.Values, key)
		delete(s.NoVal, key)
		// rebuild Keys without key
		newKeys := s.Keys[:0]
		for _, k := range s.Keys {
			if k != key {
				newKeys = append(newKeys, k)
			}
		}
		s.Keys = newKeys
		return true
	}
	if s.NoVal[key] {
		delete(s.NoVal, key)
		newKeys := s.Keys[:0]
		for _, k := range s.Keys {
			if k != key {
				newKeys = append(newKeys, k)
			}
		}
		s.Keys = newKeys
		return true
	}
	return false
}

func (s *CfgSection) Has(key string) bool {
	_, ok := s.Values[key]
	return ok || s.NoVal[key]
}

// ConfigParserObj is a Python configparser.ConfigParser/RawConfigParser instance.
type ConfigParserObj struct {
	Defaults  *CfgSection
	Sections  []string // non-DEFAULT section names in order
	Data      map[string]*CfgSection

	DefaultSection        string
	AllowNoValue          bool
	Delimiters            []string
	CommentPrefixes       []string
	InlineCommentPrefixes []string
	Strict                bool
	EmptyLinesInValues    bool
	Interpolation         int // 0=basic, 1=extended, 2=none

	// Exception classes (populated by buildConfigParser).
	NoSecErr      *Class
	DupSecErr     *Class
	DupOptErr     *Class
	NoOptErr      *Class
	InterpMissErr *Class
	InterpDepthErr *Class
	InterpSynErr  *Class
	MissSecHdrErr *Class
	ParseErr      *Class

	// BoolStates is the BOOLEAN_STATES dict (ConfigParser only, nil for Raw).
	BoolStates *Dict
}

// SectionProxyObj wraps a ConfigParserObj for mapping access to one section.
type SectionProxyObj struct {
	Parser  *ConfigParserObj
	Section string
}

// --- CSV types --------------------------------------------------------------

// CSVReader iterates rows of a csv.reader. Fields are plain Go strings.
type CSVReader struct {
	// Source is the underlying iterator yielding lines (strings). We keep
	// a reference so .__next__ can pull more lines lazily.
	Source any
	// Rows is the pre-parsed row buffer. For simplicity the stdlib layer
	// parses the whole input up-front and fills this slice.
	Rows   [][]string
	Pos    int
	LineNo int
	// Dialect holds the options that affect parsing. Exposed on the reader
	// via the `dialect` attribute.
	Dialect *CSVDialect
}

// CSVWriter writes rows to any object that has a .write(str) method.
type CSVWriter struct {
	Target  Object
	Dialect *CSVDialect
}

// CSVDictWriter wraps a CSVWriter with a fixed field order for dict rows.
type CSVDictWriter struct {
	Writer       *CSVWriter
	Fieldnames   []string
	Extrasaction string // "raise" (default) or "ignore"
}

// CSVDictReader is a proper DictReader object with fieldnames/restkey/restval.
type CSVDictReader struct {
	Rows       [][]string
	Fieldnames []string // nil = not yet loaded; read from first row on access
	Pos        int
	Dialect    *CSVDialect
	Restkey    Object // key for extra fields; default None
	Restval    Object // value for missing fields; default None
}

// CSVDialectObj wraps a CSVDialect for Python-side attribute access.
type CSVDialectObj struct {
	D *CSVDialect
}

// CSVDialect is a plain-data holder for dialect options.
type CSVDialect struct {
	Delimiter      byte
	Quotechar      byte
	Escapechar     byte
	Doublequote    bool
	SkipInitial    bool
	Strict         bool
	Lineterminator string
	Quoting        int
}

// URLParseResult is the tuple-plus-attrs result of urllib.parse.urlparse.
// Attribute access is supported via getAttr, indexed access via getitem.
type URLParseResult struct {
	Scheme, Netloc, Path, Params, Query, Fragment string
}
