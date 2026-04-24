import csv
import io

# --- constants ---

print(csv.QUOTE_MINIMAL)
print(csv.QUOTE_ALL)
print(csv.QUOTE_NONNUMERIC)
print(csv.QUOTE_NONE)
print(csv.QUOTE_STRINGS)
print(csv.QUOTE_NOTNULL)

# --- csv.Error ---

try:
    raise csv.Error("test error")
except csv.Error as e:
    print("caught Error:", e)

# --- basic reader ---

rows = list(csv.reader(["a,b,c", "1,2,3"]))
print(rows)

# quoted fields
rows = list(csv.reader(['"hello, world",b', '"say ""hi""",c']))
print(rows)

# delimiter kwarg
rows = list(csv.reader(["a|b|c", "1|2|3"], delimiter="|"))
print(rows)

# skipinitialspace
rows = list(csv.reader(["a, b, c"], skipinitialspace=True))
print(rows)

# line_num
r = csv.reader(["a,b", "c,d", "e,f"])
for row in r:
    pass
print(r.line_num)

# reader dialect attribute
r = csv.reader(["a,b"])
d = r.dialect
print(d.delimiter)
print(d.quotechar)
print(d.doublequote)
print(d.skipinitialspace)
print(repr(d.lineterminator))
print(d.quoting)

# --- basic writer ---

buf = io.StringIO()
w = csv.writer(buf)
w.writerow(["a", "b", "c"])
w.writerow([1, 2, 3])
print(buf.getvalue())

# writer dialect attribute
w2 = csv.writer(io.StringIO())
print(w2.dialect.delimiter)

# writerows
buf = io.StringIO()
w = csv.writer(buf)
w.writerows([["x", "y"], ["1", "2"]])
print(buf.getvalue())

# --- quoting modes ---

# QUOTE_ALL
buf = io.StringIO()
w = csv.writer(buf, quoting=csv.QUOTE_ALL)
w.writerow(["a", "b", "1"])
print(repr(buf.getvalue()))

# QUOTE_NONNUMERIC: quote strings, leave int/float bare
buf = io.StringIO()
w = csv.writer(buf, quoting=csv.QUOTE_NONNUMERIC)
w.writerow(["a", 1, "b"])
print(repr(buf.getvalue()))

# QUOTE_NONE with escapechar
buf = io.StringIO()
w = csv.writer(buf, quoting=csv.QUOTE_NONE, escapechar="\\")
w.writerow(["a,b", "c"])
print(repr(buf.getvalue()))

# QUOTE_STRINGS (like NONNUMERIC)
buf = io.StringIO()
w = csv.writer(buf, quoting=csv.QUOTE_STRINGS)
w.writerow(["a", 1, "b", 2.5])
print(repr(buf.getvalue()))

# QUOTE_NOTNULL: quote all non-None; None → empty unquoted
buf = io.StringIO()
w = csv.writer(buf, quoting=csv.QUOTE_NOTNULL)
w.writerow(["a", None, "b"])
print(repr(buf.getvalue()))

# --- excel-tab dialect ---

buf = io.StringIO()
w = csv.writer(buf, dialect="excel-tab")
w.writerow(["a", "b"])
print(repr(buf.getvalue()))

# --- unix dialect ---

buf = io.StringIO()
w = csv.writer(buf, dialect="unix")
w.writerow(["a", "b"])
print(repr(buf.getvalue()))

# --- DictReader ---

# basic
lines = ["name,age", "alice,30", "bob,25"]
dr = csv.DictReader(lines)
print(dr.fieldnames)
for row in dr:
    print(dict(row))

# with explicit fieldnames
lines = ["alice,30", "bob,25"]
dr = csv.DictReader(lines, fieldnames=["name", "age"])
for row in dr:
    print(dict(row))

# restval (missing field gets restval)
lines = ["a,b,c", "1,2"]
dr = csv.DictReader(lines, restval="N/A")
for row in dr:
    print(dict(row))

# restkey (extra fields go to list under restkey)
lines = ["a,b", "1,2,3,4"]
dr = csv.DictReader(lines, restkey="extras")
for row in dr:
    print(dict(row))

# --- DictWriter ---

# basic with writeheader
buf = io.StringIO()
dw = csv.DictWriter(buf, fieldnames=["name", "age"])
dw.writeheader()
dw.writerow({"name": "alice", "age": 30})
dw.writerow({"name": "bob", "age": 25})
print(buf.getvalue())

# writerows
buf = io.StringIO()
dw = csv.DictWriter(buf, fieldnames=["x", "y"])
dw.writeheader()
dw.writerows([{"x": 1, "y": 2}, {"x": 3, "y": 4}])
print(buf.getvalue())

# extrasaction='raise' (default)
try:
    buf = io.StringIO()
    dw = csv.DictWriter(buf, fieldnames=["name", "age"])
    dw.writerow({"name": "alice", "age": 30, "extra": "bad"})
except ValueError:
    print("ValueError raised")

# extrasaction='ignore'
buf = io.StringIO()
dw = csv.DictWriter(buf, fieldnames=["name", "age"], extrasaction="ignore")
dw.writerow({"name": "alice", "age": 30, "extra": "ignored"})
print(repr(buf.getvalue()))

# --- dialect registry ---

csv.register_dialect("pipes", delimiter="|")
rows = list(csv.reader(["a|b|c", "1|2|3"], "pipes"))
print(rows)

print("pipes" in csv.list_dialects())
d = csv.get_dialect("pipes")
print(d.delimiter)

csv.unregister_dialect("pipes")
print("pipes" not in csv.list_dialects())

try:
    csv.unregister_dialect("nonexistent")
except csv.Error:
    print("unregister Error raised")

# built-in dialects are always listed
all_dialects = csv.list_dialects()
print("excel" in all_dialects)
print("excel-tab" in all_dialects)
print("unix" in all_dialects)

# --- field_size_limit ---

print(csv.field_size_limit())
csv.field_size_limit(1000)
print(csv.field_size_limit())
csv.field_size_limit(131072)
print(csv.field_size_limit())

# --- Sniffer ---

sniffer = csv.Sniffer()
d = sniffer.sniff("a,b,c\n1,2,3\n")
print(d.delimiter)

d = sniffer.sniff("a|b|c\n1|2|3\n")
print(d.delimiter)

print(sniffer.has_header("name,age\nalice,30\nbob,25\n"))
print(sniffer.has_header("1,2,3\n4,5,6\n"))
