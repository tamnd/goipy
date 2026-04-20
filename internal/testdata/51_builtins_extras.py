# Long-tail builtins: pow (incl. 3-arg), format, ascii, slice, dir, delattr,
# Ellipsis / NotImplemented singletons.

# pow: 2-arg and 3-arg forms.
print(pow(2, 10))
print(pow(2, 10, 1000))
print(pow(3, -1, 7))  # modular inverse of 3 mod 7
print(pow(2.0, 3))
print(round(pow(2, 0.5), 10))

# format
print(format(255, "x"))
print(format(3.14159, ".2f"))
print(format("hi"))
print(format(42, ""))

# ascii
print(ascii("ab"))
print(ascii("cafe"))
print(ascii("café"))

# slice constructor + use as subscript.
s = slice(1, 5, 2)
print(s.start, s.stop, s.step)
print([1, 2, 3, 4, 5][s])
print("abcdef"[slice(1, None, 2)])

# Ellipsis / NotImplemented singletons.
print(...)
print(type(...).__name__)
print(... is Ellipsis)
print(NotImplemented)
print(type(NotImplemented).__name__)

# delattr removes an instance attribute.
class C:
    pass
c = C()
c.x = 1
print(hasattr(c, "x"))
delattr(c, "x")
print(hasattr(c, "x"))

# dir on a simple instance shows class + instance attrs.
class D:
    def m(self):
        pass
d = D()
d.field = 1
print([n for n in dir(d) if not n.startswith("_")])

# pow() error cases.
try:
    pow(2, 3, 0)
except ValueError:
    print("mod 0 rejected")
