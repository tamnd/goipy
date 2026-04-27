import builtins

# Module identity
print(builtins.__name__ == "builtins")   # True
print(isinstance(builtins.__doc__, str))  # True

# Built-in functions accessible through the module
print(builtins.len([1, 2, 3]) == 3)      # True
print(builtins.abs(-5) == 5)             # True
print(builtins.min(3, 1, 2) == 1)        # True
print(builtins.max(3, 1, 2) == 3)        # True
print(builtins.sum([1, 2, 3]) == 6)      # True
print(builtins.round(3.7) == 4)          # True
print(builtins.chr(65) == "A")           # True
print(builtins.ord("A") == 65)           # True
print(builtins.hex(255) == "0xff")       # True
print(builtins.oct(8) == "0o10")         # True
print(builtins.bin(10) == "0b1010")      # True

# Type constructors
print(builtins.int("42") == 42)          # True
print(builtins.str(123) == "123")        # True
print(builtins.bool(0) is False)         # True
print(builtins.bool(1) is True)          # True
print(builtins.list((1, 2)) == [1, 2])   # True
print(builtins.tuple([1, 2]) == (1, 2))  # True
print(builtins.dict(a=1) == {"a": 1})    # True
print(builtins.set([1, 2, 2]) == {1, 2}) # True

# Singletons via getattr (None/True/False are keywords)
print(getattr(builtins, "None") is None)    # True
print(getattr(builtins, "True") is True)    # True
print(getattr(builtins, "False") is False)  # True

# issubclass
print(builtins.issubclass(bool, int))    # True

# repr / ascii
print(builtins.repr(42) == "42")         # True
print(builtins.repr("hello") == "'hello'")  # True

# range / enumerate / zip / map / filter
print(list(builtins.range(3)) == [0, 1, 2])  # True
pairs = list(builtins.enumerate(["a", "b"]))
print(pairs[0] == (0, "a"))              # True
zipped = list(builtins.zip([1, 2], ["a", "b"]))
print(zipped[0] == (1, "a"))             # True
doubled = list(builtins.map(lambda x: x * 2, [1, 2, 3]))
print(doubled == [2, 4, 6])             # True
evens = list(builtins.filter(lambda x: x % 2 == 0, [1, 2, 3, 4]))
print(evens == [2, 4])                   # True

# sorted / reversed
print(builtins.sorted([3, 1, 2]) == [1, 2, 3])  # True
print(list(builtins.reversed([1, 2, 3])) == [3, 2, 1])  # True

# any / all
print(builtins.any([False, True, False]))  # True
print(builtins.all([True, True, True]))    # True
print(builtins.any([False, False]))        # False
print(builtins.all([True, False]))         # False

# Exception classes accessible
try:
    raise builtins.ValueError("test error")
except builtins.ValueError as e:
    print(str(e) == "test error")  # True

try:
    raise builtins.TypeError("wrong type")
except builtins.TypeError:
    print(True)  # True

# hasattr / getattr / setattr
class Obj:
    x = 10

o = Obj()
print(builtins.hasattr(o, "x"))           # True
print(builtins.getattr(o, "x") == 10)     # True
builtins.setattr(o, "y", 20)
print(builtins.getattr(o, "y") == 20)     # True

# dir returns a list
d = builtins.dir([])
print(isinstance(d, list))               # True

# iter / next
it = builtins.iter([10, 20])
print(builtins.next(it) == 10)            # True
print(builtins.next(it) == 20)            # True

# format
print(builtins.format(3.14, ".1f") == "3.1")  # True

# divmod
print(builtins.divmod(10, 3) == (3, 1))  # True

# pow
print(builtins.pow(2, 10) == 1024)       # True

# __import__
os_mod = builtins.__import__("os")
print(hasattr(os_mod, "path"))           # True

# The module shares the live builtin namespace — shadowing works
original_len = builtins.len
builtins.len = lambda x: 42
print(len([1, 2, 3]) == 42)             # True
builtins.len = original_len
print(len([1, 2, 3]) == 3)              # True

# exec with a code object
import marshal as _m
# We can't compile strings in goipy, but we can verify exec is present
print(callable(builtins.exec))           # True
print(callable(builtins.compile))        # True

# builtins module is the same as __builtins__ (same dict)
print(builtins.__name__ == "builtins")   # True

print("done")
