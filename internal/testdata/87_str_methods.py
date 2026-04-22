"""Tests for str methods."""

# --- case conversion ---
print("hello".upper())
print("WORLD".lower())
print("hello world".title())
print("Hello World".swapcase())
print("Hello World".casefold())
print("hello world".capitalize())

# --- predicates ---
print("abc".isalpha())
print("123".isdigit())
print("abc123".isalnum())
print("   ".isspace())
print("ABC".isupper())
print("abc".islower())
print("Hello World".istitle())
print("hello_world".isidentifier())
print("hello\x01".isprintable())
print("hello".isprintable())
print("hello".isascii())
print("héllo".isascii())
print("123".isdecimal())
print("123".isnumeric())
print("".isalpha())
print("".isupper())

# --- strip ---
print("  hello  ".strip())
print("  hello  ".lstrip())
print("  hello  ".rstrip())
print("xxhelloxx".strip("x"))
print("xxhelloxx".lstrip("x"))
print("xxhelloxx".rstrip("x"))

# --- split ---
print("a b c".split())
print("a,b,c".split(","))
print("a b c d".split(None, 2))
print("a,b,c,d".split(",", 2))
print("a b c".rsplit())
print("a,b,c".rsplit(","))
print("a,b,c,d".rsplit(",", 2))

# --- splitlines ---
print("a\nb\nc".splitlines())
print("a\r\nb\rc".splitlines())

# --- partition / rpartition ---
print("hello world".partition(" "))
print("hello world foo".rpartition(" "))
print("hello".partition("."))

# --- join ---
print(", ".join(["a", "b", "c"]))
print("".join(["x", "y", "z"]))

# --- find / rfind / index / rindex ---
print("hello".find("l"))
print("hello".rfind("l"))
print("hello".find("z"))
print("hello".index("l"))
print("hello".rindex("l"))
try:
    "hello".index("z")
except ValueError as e:
    print("ValueError")

# --- count ---
print("hello".count("l"))
print("hello".count("z"))

# --- startswith / endswith ---
print("hello".startswith("he"))
print("hello".endswith("lo"))
print("hello".startswith(("he", "wo")))
print("hello".endswith(("lo", "ld")))

# --- replace ---
print("hello".replace("l", "r"))
print("hello".replace("l", "r", 1))

# --- removeprefix / removesuffix ---
print("hello world".removeprefix("hello "))
print("hello world".removesuffix(" world"))
print("hello".removeprefix("bye"))

# --- padding ---
print("hi".center(10))
print("hi".center(10, "-"))
print("hi".ljust(10))
print("hi".ljust(10, "."))
print("hi".rjust(10))
print("hi".rjust(10, "."))
print("42".zfill(5))
print("-42".zfill(5))
print("+42".zfill(5))

# --- expandtabs ---
print("a\tb".expandtabs(4))
print("a\tb".expandtabs(8))

# --- translate / maketrans ---
t = str.maketrans("abc", "ABC")
print("abcdef".translate(t))
t2 = str.maketrans({"x": "X", "y": "Y"})
print("xyz".translate(t2))
t3 = str.maketrans("", "", "aeiou")
print("hello world".translate(t3))

# --- encode ---
print("hello".encode())
print("hello".encode("utf-8"))

# --- format ---
print("{} {}".format("hello", "world"))
print("{0} {1}".format("a", "b"))
print("{name}".format(name="Alice"))
print("{0} and {name}".format("Bob", name="Carol"))
