"""Comprehensive dataclasses test — covers all public API from the Python docs."""
import dataclasses
from dataclasses import (
    dataclass, field, fields, asdict, astuple,
    is_dataclass, replace, make_dataclass,
    MISSING, KW_ONLY, FrozenInstanceError, Field,
)


# ── __dataclass_fields__ and __dataclass_params__ direct access ───────────────

@dataclass
class Point:
    x: int
    y: int

def test_meta():
    print(isinstance(Point.__dataclass_fields__, dict))   # True
    print('x' in Point.__dataclass_fields__)               # True
    print('y' in Point.__dataclass_fields__)               # True
    p = Point.__dataclass_params__
    print(p.frozen)                                        # False
    print(p.order)                                         # False
    print(p.init)                                          # True
    print('meta ok')

test_meta()


# ── Field attributes ──────────────────────────────────────────────────────────

@dataclass
class Annotated:
    x: int
    y: str = field(default='hello', repr=True, compare=True)
    z: float = field(default=0.0, repr=False, compare=False)

def test_field_attrs():
    flds = fields(Annotated)
    print(flds[0].name)        # x
    print(flds[1].name)        # y
    print(flds[2].name)        # z
    print(flds[1].default)     # hello
    print(flds[2].default)     # 0.0
    # repr flags
    print(bool(flds[1].repr))  # True
    print(bool(flds[2].repr))  # False
    # compare flags
    print(bool(flds[1].compare))  # True
    print(bool(flds[2].compare))  # False
    # fields() on instance
    inst = Annotated(x=1)
    print(len(fields(inst)))   # 3
    print('field_attrs ok')

test_field_attrs()


# ── field(compare=False) effect on __eq__ ────────────────────────────────────

def test_compare_false():
    a = Annotated(x=1, y='hi', z=99.0)
    b = Annotated(x=1, y='hi', z=0.0)
    # z has compare=False → differs but __eq__ ignores it
    print(a == b)   # True
    c = Annotated(x=2, y='hi', z=0.0)
    print(a == c)   # False
    print('compare_false ok')

test_compare_false()


# ── field(metadata=...) ───────────────────────────────────────────────────────

@dataclass
class Measured:
    distance: float = field(metadata={'unit': 'meters', 'precision': 2})

def test_metadata():
    fld = fields(Measured)[0]
    md = fld.metadata
    print(md['unit'])       # meters
    print(md['precision'])  # 2
    print('metadata ok')

test_metadata()


# ── @dataclass(eq=False) ─────────────────────────────────────────────────────

@dataclass(eq=False)
class NoEq:
    x: int

def test_eq_false():
    a = NoEq(1)
    b = NoEq(1)
    print(a == a)   # True  (identity)
    print(a == b)   # False (different objects, no __eq__)
    print('eq_false ok')

test_eq_false()


# ── @dataclass(repr=False) ───────────────────────────────────────────────────

@dataclass(repr=False)
class NoRepr:
    x: int
    def __repr__(self):
        return f'NoRepr<{self.x}>'

def test_repr_false():
    r = repr(NoRepr(42))
    print(r)   # NoRepr<42>
    print('repr_false ok')

test_repr_false()


# ── @dataclass(init=False) ───────────────────────────────────────────────────

@dataclass(init=False)
class ManualInit:
    x: int
    def __init__(self, val):
        self.x = val * 2

def test_init_false():
    m = ManualInit(5)
    print(m.x)   # 10
    print('init_false ok')

test_init_false()


# ── unsafe_hash=True ─────────────────────────────────────────────────────────

@dataclass(unsafe_hash=True)
class Hashable:
    x: int
    y: str

def test_unsafe_hash():
    h1 = Hashable(1, 'a')
    h2 = Hashable(1, 'a')
    h3 = Hashable(2, 'b')
    print(hash(h1) == hash(h2))   # True
    print(hash(h1) == hash(h3))   # False
    print('unsafe_hash ok')

test_unsafe_hash()


# ── match_args ────────────────────────────────────────────────────────────────

@dataclass
class MatchMe:
    x: int
    y: int
    z: int = 0

def test_match_args():
    print(MatchMe.__match_args__)          # ('x', 'y', 'z')
    print(len(MatchMe.__match_args__))     # 3
    print(MatchMe.__match_args__[0])       # x
    print('match_args ok')

test_match_args()


@dataclass(match_args=False)
class NoMatch:
    x: int

def test_no_match_args():
    print(hasattr(NoMatch, '__match_args__'))   # False
    print('no_match_args ok')

test_no_match_args()


# ── kw_only=True at class level ───────────────────────────────────────────────

@dataclass(kw_only=True)
class KwOnly:
    x: int
    y: int = 0

def test_kw_only_class():
    k = KwOnly(x=5, y=10)
    print(k.x, k.y)   # 5 10
    k2 = KwOnly(x=3)
    print(k2.x, k2.y) # 3 0
    print('kw_only_class ok')

test_kw_only_class()


# ── KW_ONLY sentinel ─────────────────────────────────────────────────────────

@dataclass
class Mixed:
    a: int
    b: int = 0
    _: KW_ONLY
    c: int = 0
    d: int = 1

def test_kw_only_sentinel():
    m = Mixed(10, 20, c=30, d=40)
    print(m.a, m.b, m.c, m.d)   # 10 20 30 40
    m2 = Mixed(1)
    print(m2.a, m2.b, m2.c, m2.d)  # 1 0 0 1
    # c and d are kw_only: check __match_args__ excludes them
    print(Mixed.__match_args__)    # ('a', 'b')
    print('kw_only_sentinel ok')

test_kw_only_sentinel()


# ── Inheritance ───────────────────────────────────────────────────────────────

@dataclass
class Vehicle:
    make: str
    year: int

@dataclass
class Car(Vehicle):
    doors: int = 4

def test_inheritance():
    c = Car(make='Toyota', year=2020, doors=2)
    print(c.make, c.year, c.doors)   # Toyota 2020 2
    # fields() includes inherited fields
    flds = fields(Car)
    print(len(flds))                  # 3
    print(flds[0].name)               # make
    print(flds[1].name)               # year
    print(flds[2].name)               # doors
    # __eq__ and __repr__ work across inherited fields
    c2 = Car(make='Toyota', year=2020, doors=2)
    print(c == c2)                    # True
    print(repr(c))                    # Car(make='Toyota', year=2020, doors=4) — doors=2
    print('inheritance ok')

test_inheritance()


# ── Multi-level inheritance ───────────────────────────────────────────────────

@dataclass
class ElectricCar(Car):
    range_km: int = 400

def test_multi_inherit():
    ec = ElectricCar(make='Tesla', year=2023, doors=4, range_km=500)
    print(ec.make, ec.year, ec.doors, ec.range_km)   # Tesla 2023 4 500
    flds = fields(ElectricCar)
    print(len(flds))   # 4
    print('multi_inherit ok')

test_multi_inherit()


# ── make_dataclass with bases and extra kwargs ────────────────────────────────

def test_make_dataclass_advanced():
    # With bases
    Child = make_dataclass('Child', [('z', int)], bases=(Vehicle,))
    ch = Child(make='Ford', year=2010, z=99)
    print(ch.make, ch.year, ch.z)   # Ford 2010 99
    print(is_dataclass(Child))       # True

    # With frozen=True
    Frozen = make_dataclass('Frozen', [('val', int)], frozen=True)
    f = Frozen(val=42)
    print(f.val)                     # 42
    try:
        f.val = 99
        print('no error')
    except FrozenInstanceError:
        print('frozen ok')           # frozen ok

    print('make_dataclass_advanced ok')

test_make_dataclass_advanced()


# ── field() ValueError: both default and default_factory ─────────────────────

def test_field_both_defaults():
    try:
        field(default=1, default_factory=list)
        print('no error')
    except ValueError:
        print('ValueError raised')   # ValueError raised
    print('field_both_defaults ok')

test_field_both_defaults()


# ── MISSING sentinel identity ─────────────────────────────────────────────────

def test_missing():
    print(MISSING is not None)        # True
    print(MISSING is not MISSING)     # False
    fld = field()
    print(fld.default is MISSING)     # True
    print(fld.default_factory is MISSING)  # True
    print('missing ok')

test_missing()


# ── Field repr ────────────────────────────────────────────────────────────────

def test_field_repr():
    flds = fields(Point)
    r = repr(flds[0])
    print(isinstance(r, str))   # True
    print('x' in r)             # True
    print('field_repr ok')

test_field_repr()


# ── __dataclass_params__ frozen/order flags ───────────────────────────────────

@dataclass(frozen=True, order=True)
class Coord:
    x: float
    y: float

def test_params_flags():
    p = Coord.__dataclass_params__
    print(p.frozen)   # True
    print(p.order)    # True
    c1 = Coord(1.0, 2.0)
    c2 = Coord(1.0, 3.0)
    print(c1 < c2)    # True
    print('params_flags ok')

test_params_flags()

print("done")
