//go:build sqlite3

package vm

// Integration test for the sqlite3 module using a real SQLite driver.
//
// Run with:
//
//	go test -tags sqlite3 ./vm/ -run TestSQLite3Live -v
//
// The build tag keeps this file (and the driver import) out of the default
// build so go.mod stays dependency-free in CI.

// To run this test, register a driver before the test executes and set
// SQLite3DriverName to match. The easiest way is a local _test helper file
// (not checked in) that imports the driver and calls init:
//
//	//go:build sqlite3
//	package vm
//	import _ "modernc.org/sqlite"
//	func init() { SQLite3DriverName = "sqlite" }
//
// Then: go test -tags sqlite3 ./vm/ -run TestSQLite3Live -v

import (
	"bytes"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/goipy/marshal"
)

const sqlite3LiveScript = `import sqlite3

# ===== connect to in-memory database =====
con = sqlite3.connect(':memory:')
print(type(con).__name__)                    # Connection

# ===== basic DDL + DML =====
cur = con.cursor()
cur.execute('CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)')
cur.execute("INSERT INTO users (name, age) VALUES ('Alice', 30)")
cur.execute("INSERT INTO users (name, age) VALUES ('Bob', 25)")
con.commit()

# ===== fetchone =====
cur.execute('SELECT name, age FROM users ORDER BY age')
row = cur.fetchone()
print(row[0], row[1])                        # Bob 25

# ===== fetchall =====
cur.execute('SELECT name FROM users ORDER BY name')
rows = cur.fetchall()
print([r[0] for r in rows])                  # ['Alice', 'Bob']

# ===== rowcount =====
cur.execute("UPDATE users SET age = age + 1 WHERE age < 30")
print(cur.rowcount)                          # 1

# ===== lastrowid =====
cur.execute("INSERT INTO users (name, age) VALUES ('Carol', 41)")
print(cur.lastrowid > 0)                     # True

# ===== parameterised queries — positional =====
cur.execute('SELECT name FROM users WHERE age = ?', (41,))
row = cur.fetchone()
print(row[0])                                # Carol

# ===== executemany =====
data = [('Dave', 20), ('Eve', 22)]
cur.executemany("INSERT INTO users (name, age) VALUES (?, ?)", data)
cur.execute('SELECT count(*) FROM users')
print(cur.fetchone()[0])                     # 5

# ===== fetchmany =====
cur.execute('SELECT name FROM users ORDER BY name')
batch = cur.fetchmany(2)
print(len(batch))                            # 2
print(batch[0][0])                           # Alice

# ===== arraysize =====
cur.arraysize = 3
cur.execute('SELECT name FROM users ORDER BY name')
batch = cur.fetchmany()
print(len(batch))                            # 3

# ===== named params =====
cur.execute('SELECT name FROM users WHERE name = :n', {'n': 'Dave'})
print(cur.fetchone()[0])                     # Dave

# ===== executescript =====
cur.executescript("""
    CREATE TABLE log (msg TEXT);
    INSERT INTO log VALUES ('hello');
""")
cur.execute('SELECT msg FROM log')
print(cur.fetchone()[0])                     # hello

# ===== connection shortcut methods =====
con.execute('INSERT INTO log VALUES (?)', ('world',))
rows = con.execute('SELECT msg FROM log ORDER BY rowid').fetchall()
print([r[0] for r in rows])                  # ['hello', 'world']

# ===== OperationalError on bad SQL =====
try:
    con.execute('SELECT * FROM no_such_table')
except sqlite3.OperationalError as e:
    print('OperationalError:', 'no such table' in str(e))  # OperationalError: True

# ===== IntegrityError on constraint violation =====
con.execute('CREATE TABLE uniq (x INTEGER UNIQUE)')
con.execute('INSERT INTO uniq VALUES (1)')
try:
    con.execute('INSERT INTO uniq VALUES (1)')
except sqlite3.IntegrityError as e:
    print('IntegrityError caught')            # IntegrityError caught

# ===== cursor description =====
cur.execute('SELECT name, age FROM users LIMIT 1')
desc = cur.description
print(desc[0][0])                            # name
print(desc[1][0])                            # age

# ===== close =====
cur.close()
con.close()

# ===== basic row access =====
con2 = sqlite3.connect(':memory:')
con2.execute('CREATE TABLE t (a INTEGER, b TEXT)')
con2.execute("INSERT INTO t VALUES (1, 'x')")
cur2 = con2.execute('SELECT a, b FROM t')
row2 = cur2.fetchone()
print(row2[0], row2[1])                      # 1 x

con2.close()
print('done')
`

const sqlite3LiveExpected = `Connection
Bob 25
['Alice', 'Bob']
1
True
Carol
5
2
Alice
3
Dave
hello
['hello', 'world']
OperationalError: True
IntegrityError caught
name
age
1 x
done
`

func TestSQLite3Live(t *testing.T) {
	// Skip if no driver has been registered for SQLite3DriverName.
	registered := false
	for _, d := range sql.Drivers() {
		if d == SQLite3DriverName {
			registered = true
			break
		}
	}
	if !registered {
		t.Skipf("no SQL driver registered for %q; see file header for setup instructions", SQLite3DriverName)
	}

	// Write Python source to a temp dir, compile with python3, then load pyc.
	tmpDir, err := os.MkdirTemp("", "sqlite3_live_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pySrc := filepath.Join(tmpDir, "sqlite3_live.py")
	pyc := filepath.Join(tmpDir, "sqlite3_live.pyc")

	if err := os.WriteFile(pySrc, []byte(sqlite3LiveScript), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile with the system python3.
	out, err := exec.Command("python3", "-c",
		"import py_compile; py_compile.compile(r'"+pySrc+"', cfile=r'"+pyc+"', doraise=True)",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("py_compile failed: %v\n%s", err, out)
	}

	code, err := marshal.LoadPyc(pyc)
	if err != nil {
		t.Fatalf("LoadPyc: %v", err)
	}

	var buf bytes.Buffer
	interp := New()
	interp.Stdout = &buf
	interp.SearchPath = []string{tmpDir}

	if _, err := interp.Run(code); err != nil {
		t.Fatalf("run: %v\noutput so far:\n%s", err, buf.String())
	}

	got := buf.String()
	if got != sqlite3LiveExpected {
		wantLines := strings.Split(strings.TrimRight(sqlite3LiveExpected, "\n"), "\n")
		gotLines := strings.Split(strings.TrimRight(got, "\n"), "\n")
		maxLen := len(wantLines)
		if len(gotLines) > maxLen {
			maxLen = len(gotLines)
		}
		t.Errorf("output mismatch:")
		for idx := 0; idx < maxLen; idx++ {
			w, g := "", ""
			if idx < len(wantLines) {
				w = wantLines[idx]
			}
			if idx < len(gotLines) {
				g = gotLines[idx]
			}
			mark := " "
			if w != g {
				mark = "!"
			}
			t.Logf("%s [%3d] want=%q got=%q", mark, idx+1, w, g)
		}
	}
}
