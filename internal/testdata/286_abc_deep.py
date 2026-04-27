"""Comprehensive abc test — covers all public API from the Python docs."""
from abc import (
    ABC, ABCMeta, abstractmethod,
    abstractclassmethod, abstractstaticmethod, abstractproperty,
    get_cache_token,
)


# ── abstractmethod sets __isabstractmethod__ ──────────────────────────────────

def test_isabstractmethod():
    @abstractmethod
    def my_method(self):
        pass
    print(my_method.__isabstractmethod__)   # True

    def regular():
        pass
    print(getattr(regular, '__isabstractmethod__', False))   # False
    print('isabstractmethod ok')

test_isabstractmethod()


# ── ABC subclass with abstract method can't be instantiated ──────────────────

def test_cannot_instantiate_abstract():
    class Shape(ABC):
        @abstractmethod
        def area(self):
            pass

    try:
        s = Shape()
        print('no error')
    except TypeError as e:
        print('TypeError raised')   # TypeError raised
        print('area' in str(e))     # True
    print('cannot_instantiate ok')

test_cannot_instantiate_abstract()


# ── Concrete subclass CAN be instantiated ────────────────────────────────────

def test_concrete_ok():
    class Shape(ABC):
        @abstractmethod
        def area(self):
            pass

    class Circle(Shape):
        def area(self):
            return 3.14

    c = Circle()
    print(c.area())   # 3.14
    print('concrete_ok ok')

test_concrete_ok()


# ── Multiple abstract methods ─────────────────────────────────────────────────

def test_multiple_abstract():
    class Animal(ABC):
        @abstractmethod
        def speak(self):
            pass
        @abstractmethod
        def move(self):
            pass

    try:
        Animal()
        print('no error')
    except TypeError as e:
        msg = str(e)
        print('TypeError raised')               # TypeError raised
        print('speak' in msg or 'move' in msg)  # True
    print('multiple_abstract ok')

test_multiple_abstract()


# ── Partial implementation still abstract ────────────────────────────────────

def test_partial_abstract():
    class Animal(ABC):
        @abstractmethod
        def speak(self):
            pass
        @abstractmethod
        def move(self):
            pass

    class Dog(Animal):
        def speak(self):
            return 'woof'
        # move is still abstract

    try:
        Dog()
        print('no error')
    except TypeError:
        print('TypeError raised')   # TypeError raised
    print('partial_abstract ok')

test_partial_abstract()


# ── Full implementation works ─────────────────────────────────────────────────

def test_full_impl():
    class Animal(ABC):
        @abstractmethod
        def speak(self):
            pass
        @abstractmethod
        def move(self):
            pass

    class Cat(Animal):
        def speak(self):
            return 'meow'
        def move(self):
            return 'walk'

    c = Cat()
    print(c.speak())   # meow
    print(c.move())    # walk
    print('full_impl ok')

test_full_impl()


# ── register() makes isinstance work ─────────────────────────────────────────

def test_register():
    class MySeq(ABC):
        pass

    class NotReallyASeq:
        pass

    MySeq.register(NotReallyASeq)
    obj = NotReallyASeq()
    print(isinstance(obj, MySeq))            # True
    print('register ok')

test_register()


# ── issubclass with register ──────────────────────────────────────────────────

def test_register_subclass():
    class MyABC(ABC):
        pass

    class Unrelated:
        pass

    MyABC.register(Unrelated)
    print(issubclass(Unrelated, MyABC))   # True
    print('register_subclass ok')

test_register_subclass()


# ── get_cache_token increments on register ───────────────────────────────────

def test_cache_token():
    t1 = get_cache_token()
    print(isinstance(t1, int))   # True

    class MyABC(ABC):
        pass
    class X:
        pass
    MyABC.register(X)
    t2 = get_cache_token()
    print(t2 > t1)   # True
    print('cache_token ok')

test_cache_token()


# ── ABCMeta directly ──────────────────────────────────────────────────────────

def test_abcmeta():
    class MyInterface(metaclass=ABCMeta):
        @abstractmethod
        def do_something(self):
            pass

    try:
        MyInterface()
        print('no error')
    except TypeError:
        print('TypeError raised')   # TypeError raised

    class Impl(MyInterface):
        def do_something(self):
            return 42

    obj = Impl()
    print(obj.do_something())   # 42
    print('abcmeta ok')

test_abcmeta()


# ── isinstance / issubclass with ABC inheritance ──────────────────────────────

def test_isinstance_inheritance():
    class Vehicle(ABC):
        @abstractmethod
        def drive(self):
            pass

    class Car(Vehicle):
        def drive(self):
            return 'vroom'

    car = Car()
    print(isinstance(car, Vehicle))   # True
    print(isinstance(car, Car))       # True
    print(issubclass(Car, Vehicle))   # True
    print('isinstance_inheritance ok')

test_isinstance_inheritance()


# ── __subclasshook__ ──────────────────────────────────────────────────────────

class _SizedABC(ABC):
    @classmethod
    def __subclasshook__(cls, C):
        if hasattr(C, '__len__'):
            return True
        return NotImplemented

class _HasLen:
    def __len__(self): return 0

class _NoLen:
    pass

def test_subclasshook():
    print(issubclass(_HasLen, _SizedABC))    # True (has __len__)
    print(issubclass(_NoLen, _SizedABC))     # False (no __len__)
    print('subclasshook ok')

test_subclasshook()


# ── abstractclassmethod / abstractstaticmethod / abstractproperty stubs ───────

def test_abstract_variants():
    @abstractproperty
    def my_prop(self):
        return 0
    print(getattr(my_prop, '__isabstractmethod__', False))   # True

    @abstractclassmethod
    def my_cls(cls):
        pass
    print(getattr(my_cls, '__isabstractmethod__', False))   # True

    @abstractstaticmethod
    def my_static():
        pass
    print(getattr(my_static, '__isabstractmethod__', False))   # True
    print('abstract_variants ok')

test_abstract_variants()


# ── ABC with property ─────────────────────────────────────────────────────────

def test_abstract_property():
    class Base(ABC):
        @property
        @abstractmethod
        def value(self):
            pass

    class Concrete(Base):
        @property
        def value(self):
            return 99

    obj = Concrete()
    print(obj.value)   # 99
    print('abstract_property ok')

test_abstract_property()


print("done")
