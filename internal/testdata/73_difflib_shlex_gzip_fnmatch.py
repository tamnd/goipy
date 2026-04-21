import difflib
import shlex
import gzip
import fnmatch

# --- difflib ---

print(difflib.get_close_matches("appel", ["ape", "apple", "peach", "puppy"]))
print(difflib.get_close_matches("word", [], n=3, cutoff=0.6))
print(difflib.SequenceMatcher(None, "abc", "abd").ratio())
print(round(difflib.SequenceMatcher(None, "abcdef", "abcxef").ratio(), 4))

# ndiff produces tagged lines.
for line in difflib.ndiff(["a", "b", "c"], ["a", "c", "d"]):
    print(line)

# unified_diff with file labels.
for line in difflib.unified_diff(["a", "b"], ["a", "c"], fromfile="old.txt", tofile="new.txt"):
    print(line)

# --- shlex ---

print(shlex.quote("safe"))
print(shlex.quote("has space"))
print(shlex.quote("with'quote"))
print(shlex.quote(""))
print(shlex.join(["echo", "hello world", "x=1"]))
print(shlex.split("echo 'hello world' x=1"))
print(shlex.split('a "b c" d'))

# --- gzip ---

data = b"gzip round trip payload " * 50
c = gzip.compress(data)
print(len(c) < len(data))
print(gzip.decompress(c) == data)
# level argument also works.
print(gzip.decompress(gzip.compress(data, 9)) == data)

# --- fnmatch ---

print(fnmatch.fnmatch("hello.py", "*.py"))
print(fnmatch.fnmatch("hello.py", "*.txt"))
print(fnmatch.fnmatchcase("Foo.PY", "*.py"))
print(fnmatch.filter(["a.py", "b.txt", "c.py"], "*.py"))
print(fnmatch.fnmatch("abc", "a?c"))
print(fnmatch.fnmatch("a1c", "a[0-9]c"))
print(fnmatch.fnmatch("azc", "a[!0-9]c"))
