# V2 part 2 + V3 (roadmap 1541, spec 1544): user classes that
# subclass a builtin type can call inherited methods, take part in
# arithmetic, and round-trip through isinstance/issubclass.

print("=== int subclass ===")


class N(int):
    def doubled(self):
        return self * 2


n = N(5)
print(type(n).__name__)
print(n)
print(int(n))
print(n + 1)
print(1 + n)
print(n - 2)
print(n * 3)
print(n // 2)
print(n % 3)
print(n ** 2)
print(-n)
print(abs(N(-7)))
print(n & 6)
print(n | 2)
print(n ^ 1)
print(n << 1)
print(n >> 1)
print(n.doubled())
print(n.bit_length())
print(n == 5)
print(n == N(5))
print(n < 6)
print(isinstance(n, int))
print(isinstance(n, N))
print(issubclass(N, int))
print(issubclass(N, object))
print(N.__mro__[-1].__name__)
print({n: "a"}[5])

print("=== str subclass ===")


class S(str):
    def shout(self):
        return self + "!"


s = S("hi")
print(type(s).__name__)
print(s)
print(str(s))
print(s + " there")
print("oh " + s)
print(s * 2)
print(s.upper())
print(s.shout())
print(len(s))
print(s[0])
print(s == "hi")
print(isinstance(s, str))
print(issubclass(S, str))
print({s: 1}["hi"])

print("=== list subclass ===")


class L(list):
    def double(self):
        return list(self) + list(self)


ll = L([1, 2, 3])
print(type(ll).__name__)
print(ll)
print(ll[0])
print(ll[-1])
print(len(ll))
print(ll + [4])
print([0] + list(ll))
print(ll * 2)
print(ll.double())
print(2 in ll)
print(isinstance(ll, list))
print(issubclass(L, list))
print(sorted(ll, reverse=True))

print("=== dict subclass ===")


class D(dict):
    def keyset(self):
        return sorted(self.keys())


d = D(a=1, b=2)
print(type(d).__name__)
print(sorted(d.items()))
print(d["a"])
print(len(d))
print("a" in d)
print(d.keyset())
print(isinstance(d, dict))
print(issubclass(D, dict))

print("=== bytes subclass ===")


class B(bytes):
    pass


bb = B(b"abc")
print(type(bb).__name__)
print(bb)
print(len(bb))
print(bb[0])
print(bb + b"d")
print(bb.upper())
print(isinstance(bb, bytes))
print(issubclass(B, bytes))
