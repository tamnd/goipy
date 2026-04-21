import json
import re
import string
import copy

# --- json basics ---

# dumps scalars.
print(json.dumps(None))
print(json.dumps(True))
print(json.dumps(False))
print(json.dumps(0))
print(json.dumps(42))
print(json.dumps(-7))
print(json.dumps(3.14))
print(json.dumps("hello"))
print(json.dumps("tab\tand\nnewline"))
print(json.dumps(""))

# dumps containers.
print(json.dumps([]))
print(json.dumps({}))
print(json.dumps([1, 2, 3]))
print(json.dumps({"a": 1, "b": 2}))
print(json.dumps([1, "two", None, True]))
print(json.dumps({"nested": {"x": [1, 2]}}))

# non-ASCII: default ensure_ascii=True escapes everything above U+007F.
print(json.dumps("café"))
print(json.dumps("中文"))

# indent.
print(json.dumps([1, 2, 3], indent=2))
print(json.dumps({"a": 1, "b": 2}, indent=2, sort_keys=True))

# separators.
print(json.dumps({"a": 1, "b": 2}, separators=(",", ":")))

# sort_keys.
print(json.dumps({"b": 2, "a": 1, "c": 3}, sort_keys=True))

# loads scalars.
print(json.loads("null") is None)
print(json.loads("true"))
print(json.loads("false"))
print(json.loads("42"))
print(json.loads("3.14"))
print(json.loads('"hello"'))

# loads containers.
print(json.loads("[]"))
print(json.loads("{}"))
print(json.loads("[1,2,3]"))
print(json.loads('{"a":1}'))

# round trip.
obj = {"nums": [1, 2, 3], "name": "ok", "ok": True, "nil": None}
s = json.dumps(obj, sort_keys=True)
back = json.loads(s)
print(back == obj)

# --- re basics ---

# match anchored at start; search scans whole string.
print(re.match(r"\d+", "abc123") is None)
print(re.match(r"\w+", "abc123").group())
print(re.search(r"\d+", "abc123").group())
print(re.search(r"xyz", "abc") is None)

# fullmatch requires total match.
print(re.fullmatch(r"\d+", "123").group())
print(re.fullmatch(r"\d+", "123abc") is None)

# groups: .group(n), .groups(), .span().
m = re.search(r"(\w+)=(\d+)", "count=42")
print(m.group())
print(m.group(1))
print(m.group(2))
print(m.groups())
print(m.span())
print(m.start(1), m.end(1))

# named groups.
m = re.match(r"(?P<name>\w+)=(?P<val>\d+)", "x=9")
print(m.group("name"))
print(m.group("val"))
print(m.groupdict())

# findall: 0 groups → list of strings; 1 group → list of strings; >1 → list of tuples.
print(re.findall(r"\d+", "1,2,30,400"))
print(re.findall(r"(\w+)=(\d+)", "a=1 b=22"))

# finditer.
print([m.group() for m in re.finditer(r"\d+", "1,2,30")])

# split with and without groups.
print(re.split(r",", "a,b,c"))
print(re.split(r"(\s+)", "a  b c"))

# sub: backrefs + \g<name>.
print(re.sub(r"(\w+)=(\d+)", r"\2:\1", "x=1 y=2"))
print(re.sub(r"(?P<k>\w+)=(?P<v>\d+)", r"\g<v>:\g<k>", "x=1"))

# sub with callable.
print(re.sub(r"\d+", lambda m: str(int(m.group()) * 2), "a1 b2 c3"))

# subn returns (result, count).
print(re.subn(r"\d", "X", "a1b2c3"))

# flags.
print(re.findall(r"cat", "Cat CAT cat", re.IGNORECASE))
print(re.findall(r"^\w", "abc\ndef", re.MULTILINE))
print(re.search(r"a.b", "a\nb") is None)
print(re.search(r"a.b", "a\nb", re.DOTALL).group())

# compile → pattern methods.
p = re.compile(r"\d+")
print(p.pattern)
print(p.findall("a1b22c333"))
print(p.match("42").group())

# escape.
print(re.escape("1+2=3"))

# --- string basics ---

print(string.ascii_lowercase)
print(string.ascii_uppercase)
print(string.ascii_letters)
print(string.digits)
print(string.hexdigits)
print(string.octdigits)
print(string.punctuation)
print(string.printable[-10:])
print(" " in string.whitespace)

# --- copy basics ---

# shallow copy of a list.
a = [1, 2, [3, 4]]
b = copy.copy(a)
print(b == a)
print(a is b)
print(a[2] is b[2])  # shallow: inner list shared.

# deep copy breaks aliasing.
c = copy.deepcopy(a)
print(c == a)
print(a[2] is c[2])  # False

# mutating inner shows shallow alias.
a[2].append(5)
print(b[2])  # also [3, 4, 5]
print(c[2])  # still [3, 4]

# dict copy.
d = {"x": [1, 2], "y": 3}
d2 = copy.copy(d)
d3 = copy.deepcopy(d)
d["x"].append(9)
print(d2["x"])  # [1, 2, 9]
print(d3["x"])  # [1, 2]

# tuples (immutable) can be copied.
t = (1, [2, 3])
t2 = copy.deepcopy(t)
t[1].append(99)
print(t2[1])

# immutables short-circuit to identity (acceptable).
s = "hello"
print(copy.copy(s) == s)
print(copy.deepcopy(s) == s)
