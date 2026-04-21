import difflib
import shlex
import gzip
import fnmatch

# --- difflib scenarios -----------------------------------------------------

# 1) get_close_matches returns the closest match first.
print(difflib.get_close_matches("appel", ["apple", "ape", "puppy"]))

# 2) Exact match is returned with ratio 1.0.
print(difflib.get_close_matches("apple", ["apple", "orange"]))

# 3) cutoff filters weak matches.
print(difflib.get_close_matches("apple", ["xyz", "abc"], cutoff=0.8))

# 4) n caps the result length.
print(difflib.get_close_matches("apple", ["apple", "appe", "appl"], n=2))

# 5) Empty candidate list returns [].
print(difflib.get_close_matches("word", []))

# 6) SequenceMatcher on identical strings yields 1.0.
print(difflib.SequenceMatcher(None, "abc", "abc").ratio())

# 7) SequenceMatcher on disjoint strings yields 0.0.
print(difflib.SequenceMatcher(None, "abc", "xyz").ratio())

# 8) SequenceMatcher quick_ratio is an upper bound on ratio.
sm = difflib.SequenceMatcher(None, "abcdef", "cbadef")
print(sm.quick_ratio() >= sm.ratio())

# 9) SequenceMatcher stores a and b attributes.
sm = difflib.SequenceMatcher(None, "hi", "hey")
print(sm.a, sm.b)

# 10) set_seq1 / set_seq2 update the inputs.
sm.set_seq1("foo")
sm.set_seq2("foo")
print(sm.ratio())

# 11) ndiff with no changes returns all-context lines.
print(list(difflib.ndiff(["x", "y"], ["x", "y"])))

# 12) ndiff flags a single insertion.
print(list(difflib.ndiff(["a", "c"], ["a", "b", "c"])))

# 13) ndiff flags a single deletion.
print(list(difflib.ndiff(["a", "b", "c"], ["a", "c"])))

# 14) ndiff on empty a yields all additions.
print(list(difflib.ndiff([], ["a", "b"])))

# 15) ndiff on empty b yields all deletions.
print(list(difflib.ndiff(["a", "b"], [])))

# 16) unified_diff with headers.
for line in difflib.unified_diff(["one", "two"], ["one", "three"], fromfile="a", tofile="b"):
    print(repr(line))

# 17) unified_diff on identical inputs yields no lines.
print(list(difflib.unified_diff(["a"], ["a"])))

# 18) ratio of empty against empty is 1.0.
print(difflib.SequenceMatcher(None, "", "").ratio())

# 19) ratio scales by total length.
print(round(difflib.SequenceMatcher(None, "abcd", "abef").ratio(), 4))

# 20) get_close_matches with all matches above cutoff.
print(difflib.get_close_matches("dog", ["dog", "dag", "fog"], n=10, cutoff=0.0))

# --- shlex scenarios -------------------------------------------------------

# 21) quote passes safe chars through.
print(shlex.quote("foo"))

# 22) quote wraps empty string.
print(shlex.quote(""))

# 23) quote wraps when spaces are present.
print(shlex.quote("a b"))

# 24) quote handles embedded single quote.
print(shlex.quote("it's"))

# 25) quote handles shell metacharacters.
print(shlex.quote("a|b&c"))

# 26) quote leaves simple safe chars alone.
print(shlex.quote("path/to/file-1.txt"))

# 27) join produces a shell-safe command line.
print(shlex.join(["cmd", "arg with space", "x=1"]))

# 28) join of an empty list is empty.
print(shlex.join([]))

# 29) split reverses join for simple commands.
print(shlex.split("cmd arg1 arg2"))

# 30) split honours double quotes.
print(shlex.split('cmd "a b" c'))

# 31) split honours single quotes.
print(shlex.split("cmd 'a b' c"))

# 32) split handles backslash escapes in posix mode.
print(shlex.split("cmd a\\ b"))

# 33) split tolerates extra whitespace.
print(shlex.split("  cmd    a    b  "))

# 34) split of a single token.
print(shlex.split("only"))

# 35) split of an empty string is [].
print(shlex.split(""))

# 36) quote+split round-trip preserves the arg list.
args = ["one", "two words", "x=1", "a'b"]
rt = shlex.split(shlex.join(args))
print(rt == args)

# 37) quote an arg with a double quote.
print(shlex.quote('say "hi"'))

# 38) split with nested quotes.
print(shlex.split('''cmd "a 'b' c"'''))

# 39) quote preserves a plain identifier.
print(shlex.quote("simple_name"))

# 40) join with a single element quotes if needed.
print(shlex.join(["has space"]))

# --- gzip scenarios --------------------------------------------------------

# 41) Round-trip basic payload.
data = b"gzip test " * 20
print(gzip.decompress(gzip.compress(data)) == data)

# 42) Empty payload round trips.
print(gzip.decompress(gzip.compress(b"")) == b"")

# 43) Level 1 round trip.
print(gzip.decompress(gzip.compress(data, 1)) == data)

# 44) Level 9 round trip.
print(gzip.decompress(gzip.compress(data, 9)) == data)

# 45) Level 0 round trip.
print(gzip.decompress(gzip.compress(data, 0)) == data)

# 46) compress yields a gzip magic header (1f 8b).
out = gzip.compress(b"x")
print(out[0], out[1])

# 47) decompress tolerates bytes-like input.
print(gzip.decompress(bytearray(gzip.compress(b"abc"))) == b"abc")

# 48) Larger payload compresses to smaller output.
big = b"A" * 1000
print(len(gzip.compress(big)) < len(big))

# 49) decompress on garbage raises.
try:
    gzip.decompress(b"not a gzip stream")
    print("no error")
except Exception:
    print("decompress raised")

# 50) compresslevel kwarg works.
print(gzip.decompress(gzip.compress(data, compresslevel=6)) == data)

# --- fnmatch scenarios -----------------------------------------------------

# 51) Simple *.py glob matches.
print(fnmatch.fnmatch("test.py", "*.py"))

# 52) Wrong extension does not match.
print(fnmatch.fnmatch("test.txt", "*.py"))

# 53) ? matches one char.
print(fnmatch.fnmatch("abc", "a?c"))

# 54) [] character class.
print(fnmatch.fnmatch("a1c", "a[0-9]c"))

# 55) Negated class with !.
print(fnmatch.fnmatch("abc", "a[!0-9]c"))

# 56) Star at the start.
print(fnmatch.fnmatch("hello.py", "*.py"))

# 57) Star alone matches everything.
print(fnmatch.fnmatch("anything", "*"))

# 58) Empty pattern matches only empty string.
print(fnmatch.fnmatch("", ""))
print(fnmatch.fnmatch("x", ""))

# 59) fnmatchcase is case-sensitive.
print(fnmatch.fnmatchcase("Foo.PY", "*.py"))
print(fnmatch.fnmatchcase("Foo.py", "*.py"))

# 60) fnmatch is case-insensitive (on most platforms — we emulate POSIX).
print(fnmatch.fnmatch("Foo.PY", "*.py"))

# 61) filter returns only matching names.
print(fnmatch.filter(["a.py", "b.txt", "c.py", "d.md"], "*.py"))

# 62) filter on empty list is empty.
print(fnmatch.filter([], "*"))

# 63) filter with a class pattern.
print(fnmatch.filter(["x1", "x2", "xa", "xb"], "x[0-9]"))

# 64) translate returns a regex-ish pattern.
t = fnmatch.translate("*.py")
print(".py" in t or "\\.py" in t)

# 65) Multi-char wildcard.
print(fnmatch.fnmatch("long_name.log", "long_*.log"))

# 66) Exact literal match.
print(fnmatch.fnmatch("exact", "exact"))

# 67) Question marks don't match empty.
print(fnmatch.fnmatch("ab", "a?"))
print(fnmatch.fnmatch("a", "a?"))

# 68) Range inside brackets.
print(fnmatch.fnmatch("z", "[a-z]"))
print(fnmatch.fnmatch("Z", "[a-z]"))

# --- cross-module scenarios ------------------------------------------------

# 69) Gzip-compress then hex-ish length comparison.
c1 = gzip.compress(b"abcdef" * 100)
c2 = gzip.compress(b"ababab" * 100)
print(len(c1) < 200 and len(c2) < 200)

# 70) fnmatch.filter over strings produced by shlex.split.
names = shlex.split("a.py b.txt c.py d.md")
print(fnmatch.filter(names, "*.py"))

# 71) quote a filename that looks like a shell glob.
print(shlex.quote("*.py"))

# 72) get_close_matches picks the best of a large candidate list.
opts = ["alpha", "beta", "gamma", "alfa", "apple"]
print(difflib.get_close_matches("alph", opts, n=2))

# 73) translate a pattern and check it contains an escaped dot.
print("\\." in fnmatch.translate("*.py") or "\\\\." in fnmatch.translate("*.py"))

# 74) ndiff produces 2-prefixed lines only.
for line in difflib.ndiff(["x"], ["y"]):
    print(line[:2] in ("  ", "- ", "+ ", "? "))

# 75) shlex.join handles empty string.
print(shlex.join([""]))

# 76) fnmatch matches against filter's output.
out = fnmatch.filter(["a.py"], "*.py")
print(all(fnmatch.fnmatch(n, "*.py") for n in out))

# 77) Gzip round-trip of shlex-quoted args.
payload = shlex.join(["a", "b c", "d"]).encode("ascii", errors="replace") if False else b"a 'b c' d"
print(gzip.decompress(gzip.compress(payload)) == payload)

# 78) SequenceMatcher on diff-like input.
sm = difflib.SequenceMatcher(None, "diff me", "diff you")
print(sm.ratio() > 0.5)

# 79) get_close_matches ignores exact-but-cutoff-below candidates when cutoff=1.0.
print(difflib.get_close_matches("apple", ["apple!", "apple"], cutoff=1.0))

# 80) fnmatchcase survives a bracketed pattern.
print(fnmatch.fnmatchcase("a1", "a[1-3]"))
print(fnmatch.fnmatchcase("a4", "a[1-3]"))
