import json
import re
import string
import copy

# --- json stress ---

# 1) empty container round-trips.
print(json.dumps([]))
print(json.dumps({}))
print(json.loads("[]"))
print(json.loads("{}"))

# 2) deeply nested lists.
nested = [[[[[1]]]]]
print(json.dumps(nested))
print(json.loads(json.dumps(nested)) == nested)

# 3) mixed-type list.
m = [None, True, False, 0, 1.5, "x", [1, 2], {"k": "v"}]
print(json.loads(json.dumps(m)) == m)

# 4) integer-valued floats keep a ".0".
print(json.dumps(3.0))
print(json.dumps(1e2))

# 5) string with every JSON escape.
s = "\"\\\n\r\t\b\f"
enc = json.dumps(s)
print(json.loads(enc) == s)

# 6) surrogate pair (non-BMP).
s = "\U0001F600"
enc = json.dumps(s)
print(enc)
print(json.loads(enc) == s)

# 7) nested dict with sort_keys stable ordering.
d = {"b": 1, "a": 2, "c": {"y": 1, "x": 2}}
print(json.dumps(d, sort_keys=True))

# 8) indent=0 still newline-separates.
print(json.dumps([1, 2], indent=0))

# 9) indent=4 pretty-print.
print(json.dumps({"a": [1, 2]}, indent=4, sort_keys=True))

# 10) non-ASCII with default ensure_ascii=True.
print(json.dumps("α"))      # \u03b1
print(json.dumps("é"))      # \u00e9

# 11) integer keys coerce to string keys.
print(json.dumps({1: "a", 2: "b"}, sort_keys=True))

# 12) loads integer vs float disambiguation.
print(type(json.loads("1")).__name__)
print(type(json.loads("1.0")).__name__)

# 13) loads rejects invalid JSON.
try:
    json.loads("{broken")
except ValueError:
    print("loads-bad: ValueError")

# 14) loads with trailing whitespace.
print(json.loads("  42  "))

# 15) custom separators, no spaces.
print(json.dumps([1, 2, 3], separators=(",", ":")))

# --- re stress ---

# 16) empty pattern matches at every position but returns empty string.
m = re.match(r"", "abc")
print(m.group())
print(m.span())

# 17) match anchored: only at start.
print(re.match(r"b", "abc") is None)

# 18) search on empty string.
print(re.search(r".", "") is None)

# 19) fullmatch on empty pattern + empty string.
m = re.fullmatch(r"", "")
print(m.group())

# 20) group-zero == whole match.
m = re.search(r"(\d+)-(\d+)", "a 1-2 b")
print(m.group(0), m.group(1), m.group(2))

# 21) groups() with default for missing optional.
m = re.match(r"(\d+)(?:-(\d+))?", "42")
print(m.groups())
print(m.groups("MISSING"))

# 22) lastindex.
m = re.search(r"(a)(b)(c)", "abc")
print(m.lastindex)

# 23) findall with no match.
print(re.findall(r"\d+", "abc"))

# 24) findall with one group at many positions.
print(re.findall(r"(\w)(\w)", "abcdef"))

# 25) finditer emits Match objects.
it = re.finditer(r"\d+", "1 22 333")
spans = [m.span() for m in it]
print(spans)

# 26) split with maxsplit.
print(re.split(r",", "a,b,c,d", maxsplit=2))

# 27) split keeps empty separator pieces.
print(re.split(r"-", "--a--"))

# 28) sub with count.
print(re.sub(r"\d", "X", "1234", count=2))

# 29) sub with \\\\ literal backslash.
print(re.sub(r"x", r"\\", "ax"))

# 30) sub with \n and \t literal escapes.
print(repr(re.sub(r"x", r"\n\t", "x")))

# 31) sub with named group.
print(re.sub(r"(?P<d>\d+)", r"[\g<d>]", "a1b22"))

# 32) sub callable that returns the match uppercased.
print(re.sub(r"[a-z]+", lambda m: m.group().upper(), "abc 123 def"))

# 33) subn returns count of 0 when no match.
print(re.subn(r"\d", "X", "abc"))

# 34) flag combinations: IGNORECASE + MULTILINE.
print(re.findall(r"^[a-z]", "A\nb\nC", re.IGNORECASE | re.MULTILINE))

# 35) DOTALL lets . match newline.
print(re.findall(r".+", "a\nb", re.DOTALL))

# 36) escape quotes regex metacharacters.
print(re.escape("a.b*c?"))

# 37) compile then reuse across inputs.
p = re.compile(r"\b\w+\b")
print(p.findall("hello world"))
print(p.findall("foo-bar baz"))

# 38) Pattern.sub and Pattern.subn.
p = re.compile(r"\d")
print(p.sub("X", "a1b2"))
print(p.subn("X", "a1b2"))

# 39) invalid pattern raises.
try:
    re.compile(r"[")
except Exception as e:
    print("re-bad:", type(e).__name__ in ("error", "PatternError", "ValueError"))

# 40) Match.expand with backreferences.
m = re.search(r"(\w+)=(\d+)", "k=9")
print(m.expand(r"\2:\1"))

# 41) named group span.
m = re.search(r"(?P<k>\w+)=(?P<v>\d+)", "abc=123")
print(m.span("k"))
print(m.span("v"))

# --- string stress ---

# 42) ascii_letters = lowercase + uppercase.
print(string.ascii_letters == string.ascii_lowercase + string.ascii_uppercase)

# 43) digit and hex content.
print("0" in string.digits)
print("f" in string.hexdigits and "F" in string.hexdigits)

# 44) printable contains digits + letters + punctuation + whitespace.
print(all(c in string.printable for c in string.digits))
print(all(c in string.printable for c in string.ascii_letters))

# 45) whitespace includes tab and newline.
print("\t" in string.whitespace)
print("\n" in string.whitespace)

# --- copy stress ---

# 46) deepcopy preserves nested structure.
orig = {"a": [1, 2, {"b": [3, 4]}]}
dup = copy.deepcopy(orig)
print(dup == orig)
print(dup is not orig)
print(dup["a"][2] is not orig["a"][2])

# 47) deepcopy a set.
s = {1, 2, 3}
s2 = copy.deepcopy(s)
print(s == s2)
s.add(4)
print(3 in s2 and 4 not in s2)

# 48) deepcopy a tuple of lists.
t = ([1, 2], [3, 4])
t2 = copy.deepcopy(t)
t[0].append(99)
print(t2[0] == [1, 2])

# 49) copy of dict doesn't share top-level.
d = {"a": 1}
d2 = copy.copy(d)
d["b"] = 2
print("b" not in d2)

# 50) deepcopy of list of lists creates fresh inner lists.
xs = [[1], [2], [3]]
ys = copy.deepcopy(xs)
xs[0].append(99)
print(ys[0] == [1])

# 51) shallow vs deep distinction on dict-of-list.
d = {"x": [1, 2]}
sc = copy.copy(d)
dc = copy.deepcopy(d)
d["x"].append(3)
print(sc["x"])  # [1, 2, 3] — shared
print(dc["x"])  # [1, 2]

# 52) copy preserves element count.
xs = list(range(100))
ys = copy.copy(xs)
print(len(ys) == 100)
print(ys == xs)
print(ys is not xs)

# 53) deepcopy empty container.
print(copy.deepcopy([]))
print(copy.deepcopy({}))

# 54) copy of int is identical (immutable scalar).
print(copy.copy(42) == 42)
print(copy.deepcopy(42) == 42)

# 55) deepcopy of deeply nested structure.
deep = [1, [2, [3, [4, [5]]]]]
dup = copy.deepcopy(deep)
print(dup == deep)
deep[1][1][1][1].append(99)
print(dup[1][1][1][1] == [5])
