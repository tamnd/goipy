import dataclasses
from dataclasses import dataclass, field, fields, asdict, astuple, is_dataclass, replace, make_dataclass, MISSING, KW_ONLY, FrozenInstanceError


# ── basic dataclass ───────────────────────────────────────────────────────────

@dataclass
class Point:
    x: int
    y: int


def test_basic():
    p = Point(1, 2)
    print(p.x, p.y)
    print(repr(p))
    print(p == Point(1, 2))
    print(p == Point(1, 3))
    print(is_dataclass(p))
    print(is_dataclass(Point))
    print('basic ok')


# ── default values ────────────────────────────────────────────────────────────

@dataclass
class Config:
    host: str = 'localhost'
    port: int = 8080
    debug: bool = False


def test_defaults():
    c = Config()
    print(c.host, c.port, c.debug)
    c2 = Config(host='example.com', port=443)
    print(c2.host, c2.port, c2.debug)
    print(repr(c))
    print('defaults ok')


# ── field() with default_factory ──────────────────────────────────────────────

@dataclass
class Container:
    items: list = field(default_factory=list)
    name: str = 'unnamed'


def test_field_factory():
    c1 = Container()
    c2 = Container()
    c1.items.append(1)
    print(c1.items)
    print(c2.items)  # independent list
    print(c1.name)
    print('field_factory ok')


# ── fields() function ─────────────────────────────────────────────────────────

def test_fields():
    flds = fields(Point)
    print(len(flds))
    print(flds[0].name)
    print(flds[1].name)
    print(type(flds).__name__)
    print('fields ok')


# ── asdict() and astuple() ────────────────────────────────────────────────────

@dataclass
class Color:
    r: int
    g: int
    b: int


def test_asdict():
    c = Color(255, 128, 0)
    d = asdict(c)
    print(type(d).__name__)
    print(d['r'], d['g'], d['b'])
    print('asdict ok')


def test_astuple():
    c = Color(1, 2, 3)
    t = astuple(c)
    print(type(t).__name__)
    print(t)
    print('astuple ok')


# ── nested dataclass ──────────────────────────────────────────────────────────

@dataclass
class Line:
    start: Point
    end: Point


def test_nested():
    line = Line(Point(0, 0), Point(3, 4))
    print(repr(line.start))
    print(repr(line.end))
    d = asdict(line)
    print(type(d['start']).__name__)
    print(d['start']['x'], d['start']['y'])
    print('nested ok')


# ── replace() ────────────────────────────────────────────────────────────────

def test_replace():
    p = Point(1, 2)
    p2 = replace(p, y=10)
    print(p.x, p.y)       # original unchanged
    print(p2.x, p2.y)     # new values
    print(p2 == Point(1, 10))
    print('replace ok')


# ── frozen dataclass ──────────────────────────────────────────────────────────

@dataclass(frozen=True)
class ImmutablePoint:
    x: int
    y: int


def test_frozen():
    p = ImmutablePoint(3, 4)
    print(p.x, p.y)
    print(hash(p) == hash(ImmutablePoint(3, 4)))
    try:
        p.x = 99
        print('should not reach')
    except FrozenInstanceError:
        print('frozen error caught')
    print('frozen ok')


# ── ordering ──────────────────────────────────────────────────────────────────

@dataclass(order=True)
class Score:
    value: int


def test_order():
    a = Score(1)
    b = Score(2)
    print(a < b)
    print(b > a)
    print(a <= Score(1))
    print(b >= Score(3))
    print('order ok')


# ── __post_init__ ─────────────────────────────────────────────────────────────

@dataclass
class Circle:
    radius: float
    area: float = field(init=False)

    def __post_init__(self):
        self.area = 3.14159 * self.radius * self.radius


def test_post_init():
    c = Circle(radius=2.0)
    print(c.radius)
    print(round(c.area, 3))
    print('post_init ok')


# ── field repr=False ──────────────────────────────────────────────────────────

@dataclass
class Secret:
    username: str
    password: str = field(repr=False)


def test_field_repr_false():
    s = Secret('admin', 'hunter2')
    print(repr(s))
    print('field_repr_false ok')


# ── make_dataclass ────────────────────────────────────────────────────────────

def test_make_dataclass():
    Dyn = make_dataclass('Dyn', [('x', int), ('y', int, 0)])
    d = Dyn(x=5)
    print(d.x, d.y)
    print(is_dataclass(Dyn))
    print('make_dataclass ok')


# ── MISSING / is_dataclass checks ────────────────────────────────────────────

def test_misc():
    print(MISSING is not None)
    print(is_dataclass(42))
    print(is_dataclass(int))
    fld = field(default=42)
    print(fld.default)
    print('misc ok')


test_basic()
test_defaults()
test_field_factory()
test_fields()
test_asdict()
test_astuple()
test_nested()
test_replace()
test_frozen()
test_order()
test_post_init()
test_field_repr_false()
test_make_dataclass()
test_misc()
