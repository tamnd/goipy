# Stress for long-tail builtins: more format specs, pow edge cases,
# ascii with containers, slice/Ellipsis/NotImplemented corners, dir on
# inheritance, delattr errors.

# --- format: integer specs ---
print(format(42, "b"))
print(format(42, "o"))
print(format(255, "X"))
print(format(42, "#b"))
print(format(42, "#o"))
print(format(255, "#x"))
print(format(42, "08b"))
print(format(42, "+d"))
print(format(-42, "+d"))
print(format(42, " d"))
print(format(1234567, ","))
print(format(1234567, "_"))
print(format(42, ">10"))
print(format(42, "<10d"))
print(format(42, "^10d"))
print(format(42, "*^10d"))

# --- format: float specs ---
print(format(3.14159, ".4f"))
print(format(1234567.89, ",.2f"))
print(format(0.000123, ".2e"))
print(format(1.5, "+.2f"))
print(format(3.14, "10.2f"))
print(format(3.14, "010.2f"))

# --- format: string specs ---
print(format("hi", ">6"))
print(format("hi", "<6"))
print(format("hi", "^6"))
print(format("hi", "*^6"))

# --- pow edges ---
print(pow(10, 0))            # 1
print(pow(0, 0))             # 1
print(pow(2, 100))            # big int
print(pow(3, 100, 1000000007))
print(pow(7, -1, 11))         # modular inverse
print(pow(2, -2))             # -> 0.25 float
print(pow(-2, 3))
print(round(pow(2, 0.5) ** 2, 6))

# pow(big, big, modulus)
print(pow(2, 1000, 10**9 + 7))

# --- ascii ---
print(ascii([1, "a", "é"]))
print(ascii({"k": "ü"}))
print(ascii(("x", "π")))
print(ascii("\n\t"))

# --- slice in more forms ---
print(slice(5).start, slice(5).stop, slice(5).step)
print(slice(1, None).stop)
print("abcdefgh"[slice(None, None, 2)])
print((10, 20, 30, 40, 50)[slice(-2, None)])

# --- Ellipsis / NotImplemented ---
print(Ellipsis is ...)
print(NotImplemented is NotImplemented)
print(type(Ellipsis).__name__)
print(type(NotImplemented).__name__)

# --- dir on inheritance ---
class A:
    def a_method(self):
        pass

class B(A):
    def b_method(self):
        pass

b = B()
b.x = 1
names = [n for n in dir(b) if not n.startswith("_")]
print(sorted(names))

# --- delattr on missing attribute ---
class C:
    pass

c = C()
try:
    delattr(c, "nope")
except AttributeError:
    print("missing attr rejected")

# pow(2, 3, 1) always 0
print(pow(2, 3, 1))
