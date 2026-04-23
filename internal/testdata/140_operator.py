import operator

# ===== Arithmetic =====
print(operator.add(3, 4))           # 7
print(operator.sub(10, 3))          # 7
print(operator.mul(3, 4))           # 12
print(operator.truediv(7, 2))       # 3.5
print(operator.floordiv(7, 2))      # 3
print(operator.mod(10, 3))          # 1
print(operator.pow(2, 10))          # 1024
print(operator.neg(-5))             # 5
print(operator.pos(-3))             # -3
print(operator.abs(-7))             # 7
print(operator.abs(3.14))           # 3.14
print(operator.abs(-2+0j))          # 2.0

# ===== Bitwise =====
print(operator.and_(0b1100, 0b1010))    # 8
print(operator.or_(0b1100, 0b1010))     # 14
print(operator.xor(0b1100, 0b1010))     # 6
print(operator.lshift(1, 4))            # 16
print(operator.rshift(32, 2))           # 8
print(operator.inv(5))                  # -6
print(operator.invert(-1))              # 0

# ===== Logical =====
print(operator.not_(0))             # True
print(operator.not_(42))            # False
print(operator.truth(0))            # False
print(operator.truth('hello'))      # True

# ===== index =====
print(operator.index(5))            # 5
print(operator.index(True))         # 1

# ===== Comparisons =====
print(operator.lt(1, 2))            # True
print(operator.le(2, 2))            # True
print(operator.eq(3, 3))            # True
print(operator.ne(3, 4))            # True
print(operator.gt(5, 3))            # True
print(operator.ge(3, 3))            # True

# ===== is_ / is_not =====
a = [1, 2, 3]
b = a
c = [1, 2, 3]
print(operator.is_(a, b))           # True
print(operator.is_(a, c))           # False
print(operator.is_not(a, c))        # True
print(operator.is_(None, None))     # True

# ===== concat =====
print(operator.concat([1, 2], [3, 4]))   # [1, 2, 3, 4]
print(operator.concat('hello', ' world')) # hello world
print(operator.concat((1, 2), (3, 4)))   # (1, 2, 3, 4)

# ===== contains / getitem / setitem / delitem =====
lst = [10, 20, 30]
print(operator.contains(lst, 20))       # True
print(operator.contains(lst, 99))       # False
print(operator.getitem(lst, 1))         # 20
operator.setitem(lst, 0, 99)
print(lst)                              # [99, 20, 30]
operator.delitem(lst, 1)
print(lst)                              # [99, 30]

d = {'x': 1, 'y': 2}
print(operator.getitem(d, 'x'))         # 1
operator.setitem(d, 'z', 3)
print(d['z'])                           # 3
operator.delitem(d, 'x')
print('x' in d)                         # False

# ===== length_hint =====
print(operator.length_hint([1, 2, 3]))          # 3
print(operator.length_hint('hello'))            # 5
print(operator.length_hint({}, 42))             # 0 (empty dict)

# ===== countOf / indexOf =====
data = [1, 2, 3, 2, 1, 2]
print(operator.countOf(data, 2))        # 3
print(operator.countOf(data, 5))        # 0
print(operator.indexOf(data, 3))        # 2
print(operator.indexOf(data, 2))        # 1
try:
    operator.indexOf(data, 99)
except ValueError:
    print('indexOf not found ok')       # indexOf not found ok

# ===== In-place operators =====
# iadd: list extends in place
lst = [1, 2]
result = operator.iadd(lst, [3, 4])
print(result)                           # [1, 2, 3, 4]
print(result is lst)                    # True (same list, mutated)

# iadd: int falls back to add
print(operator.iadd(5, 3))             # 8

# iconcat: same as iadd for lists
lst2 = [10, 20]
result2 = operator.iconcat(lst2, [30, 40])
print(result2)                          # [10, 20, 30, 40]
print(result2 is lst2)                  # True

# ifloordiv
print(operator.ifloordiv(10, 3))       # 3

# ilshift
print(operator.ilshift(4, 2))          # 16

# imod
print(operator.imod(10, 3))            # 1

# imul
print(operator.imul(3, 4))             # 12

# ior
print(operator.ior(0b1010, 0b0101))    # 15

# ipow
print(operator.ipow(2, 8))             # 256

# irshift
print(operator.irshift(64, 3))         # 8

# isub
print(operator.isub(10, 4))            # 6

# itruediv
print(operator.itruediv(7, 2))         # 3.5

# ixor
print(operator.ixor(0b1111, 0b0101))   # 10

# iand
print(operator.iand(0b1110, 0b0110))   # 6

# ===== attrgetter =====
class Point:
    def __init__(self, x, y):
        self.x = x
        self.y = y

p = Point(3, 4)
get_x = operator.attrgetter('x')
get_y = operator.attrgetter('y')
print(get_x(p))                          # 3
print(get_y(p))                          # 4

# Multiple attributes
get_xy = operator.attrgetter('x', 'y')
print(get_xy(p))                         # (3, 4)

# Dotted path
class Line:
    def __init__(self, start, end):
        self.start = start
        self.end = end

line = Line(Point(0, 0), Point(5, 5))
get_end_x = operator.attrgetter('end.x')
print(get_end_x(line))                   # 5

# attrgetter with sorted
points = [Point(3, 1), Point(1, 4), Point(2, 2)]
print([p.x for p in sorted(points, key=operator.attrgetter('x'))])  # [1, 2, 3]

# ===== itemgetter =====
get0 = operator.itemgetter(0)
get1 = operator.itemgetter(1)
print(get0([10, 20, 30]))               # 10
print(get1('hello'))                    # e

# Multiple items
get01 = operator.itemgetter(0, 2)
print(get01([10, 20, 30]))             # (10, 30)

# itemgetter with dict
get_name = operator.itemgetter('name')
people = [{'name': 'Bob', 'age': 30}, {'name': 'Alice', 'age': 25}]
print([get_name(p) for p in people])    # ['Bob', 'Alice']
print(sorted(people, key=get_name)[0]['name'])  # Alice

# ===== methodcaller =====
mc_upper = operator.methodcaller('upper')
print(mc_upper('hello'))                # HELLO

mc_replace = operator.methodcaller('replace', 'l', 'r')
print(mc_replace('hello'))              # herro

# methodcaller with keyword args
mc_join = operator.methodcaller('join', ['a', 'b', 'c'])
print(mc_join('-'))                     # a-b-c

# ===== Dunder aliases =====
print(operator.__add__(2, 3))           # 5
print(operator.__mul__(3, 4))           # 12
print(operator.__neg__(-5))             # 5
print(operator.__lt__(1, 2))            # True
print(operator.__abs__(-7))             # 7
print(operator.__inv__(5))              # -6
print(operator.__invert__(-1))          # 0
print(operator.__not__(0))              # True
print(operator.__and__(12, 10))         # 8
print(operator.__or__(12, 10))           # 14
print(operator.__concat__([1], [2]))    # [1, 2]
print(operator.__contains__([1,2,3], 2))  # True
print(operator.__getitem__([10,20,30], 1)) # 20
print(operator.__iadd__(5, 3))          # 8

print('done')
