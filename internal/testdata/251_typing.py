import typing
from typing import (
    TYPE_CHECKING, cast, overload, final,
    TypeVar, Generic, Optional, Union,
    List, Dict, Tuple, get_origin, get_args,
    NamedTuple, TypedDict, is_typeddict,
    NewType, runtime_checkable, Protocol,
)


def test_constants():
    print(TYPE_CHECKING)
    print(typing.Any is not None)
    print('test_constants ok')


def test_cast():
    print(cast(int, 'hello'))
    print(cast(str, 42))
    print('test_cast ok')


def test_typevar():
    T = TypeVar('T')
    print(T.__name__)
    print(T.__constraints__)
    print(T.__bound__ is None)
    S = TypeVar('S', int, str)
    print(S.__name__)
    print(len(S.__constraints__))
    print(S.__constraints__[0] is int)
    print(S.__constraints__[1] is str)
    print('test_typevar ok')


def test_generics():
    li = List[int]
    print(get_origin(li) is list)
    print(get_args(li) == (int,))

    di = Dict[str, int]
    print(get_origin(di) is dict)
    print(get_args(di) == (str, int))

    ti = Tuple[int, str]
    print(get_origin(ti) is tuple)
    print(get_args(ti) == (int, str))
    print('test_generics ok')


def test_optional_union():
    oi = Optional[int]
    print(get_origin(oi) is Union)
    args = get_args(oi)
    print(args[0] is int)
    print(isinstance(None, args[1]))

    ui = Union[int, str]
    print(get_origin(ui) is Union)
    print(get_args(ui) == (int, str))
    print('test_optional_union ok')


def test_generic_class():
    T = TypeVar('T')

    class Stack(Generic[T]):
        def __init__(self):
            self._items = []

        def push(self, item):
            self._items.append(item)

        def pop(self):
            return self._items.pop()

    s = Stack()
    s.push(1)
    s.push(2)
    print(s.pop())
    print(s.pop())
    print('test_generic_class ok')


def test_namedtuple_functional():
    Point = NamedTuple('Point', [('x', int), ('y', int)])
    p = Point(1, 2)
    print(p.x, p.y)
    print(p[0], p[1])
    print(Point._fields)
    print('test_namedtuple_functional ok')


def test_namedtuple_class():
    class Color(NamedTuple):
        r: int
        g: int
        b: int

    c = Color(255, 0, 128)
    print(c.r, c.g, c.b)
    print(c[0])
    print(Color._fields)
    print('test_namedtuple_class ok')


def test_typeddict():
    class Movie(TypedDict):
        name: str
        year: int

    m = Movie(name='Blade Runner', year=1982)
    print(m['name'])
    print(m['year'])
    print(is_typeddict(Movie))
    print(is_typeddict(dict))
    print('test_typeddict ok')


def test_newtype():
    UserId = NewType('UserId', int)
    u = UserId(42)
    print(u)
    print(type(u).__name__)
    print('test_newtype ok')


def test_decorators():
    @overload
    def process(x: int) -> str: ...
    @overload
    def process(x: str) -> int: ...
    def process(x):
        if isinstance(x, int):
            return str(x)
        return len(x)

    print(process(42))
    print(process('hello'))

    class Base:
        @final
        def method(self):
            return 'final'

    b = Base()
    print(b.method())
    print(Base.method.__final__)
    print('test_decorators ok')


def test_protocol():
    @runtime_checkable
    class Drawable(Protocol):
        def draw(self) -> None: ...

    class Circle:
        def draw(self):
            print('drawing circle')

    c = Circle()
    print(isinstance(c, Drawable))
    c.draw()
    print('test_protocol ok')


test_constants()
test_cast()
test_typevar()
test_generics()
test_optional_union()
test_generic_class()
test_namedtuple_functional()
test_namedtuple_class()
test_typeddict()
test_newtype()
test_decorators()
test_protocol()
