package object

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
