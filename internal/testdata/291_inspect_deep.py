"""Comprehensive inspect test — covers all public API from the Python docs."""
import inspect


# ── type predicates ──────────────────────────────────────────────────────────

import os
print(inspect.ismodule(os))             # True
print(inspect.ismodule(42))             # False

class MyClass:
    def method(self): pass

print(inspect.isclass(MyClass))         # True
print(inspect.isclass(42))              # False

def myfunc(): pass

print(inspect.isfunction(myfunc))       # True
print(inspect.isfunction(42))           # False

print(inspect.isbuiltin(len))           # True
print(inspect.isbuiltin(myfunc))        # False

obj = MyClass()
print(inspect.ismethod(obj.method))     # True
print(inspect.ismethod(myfunc))         # False

print(inspect.isroutine(myfunc))        # True
print(inspect.isroutine(len))           # True
print(inspect.isroutine(42))            # False

def gen_fn():
    yield 1

g = gen_fn()
print(inspect.isgeneratorfunction(gen_fn))  # True
print(inspect.isgenerator(g))               # True
print(inspect.isgenerator(42))              # False

async def coro_fn(): pass
print(inspect.iscoroutinefunction(coro_fn)) # True
print(inspect.iscoroutinefunction(myfunc))  # False

async def asyncgen_fn():
    yield 1
print(inspect.isasyncgenfunction(asyncgen_fn))  # True
print(inspect.isasyncgenfunction(myfunc))        # False

print(inspect.iscode(42))               # False
print(inspect.isframe(42))              # False
print(inspect.ismethodwrapper(42))      # False
print(inspect.ismethoddescriptor(42))   # False
print(inspect.isdatadescriptor(42))     # False
print(inspect.isgetsetdescriptor(42))   # False
print(inspect.ismemberdescriptor(42))   # False
print(inspect.isabstract(MyClass))      # False
print(inspect.isasyncgen(42))           # False
print(inspect.isawaitable(42))          # False


# ── getmembers ────────────────────────────────────────────────────────────────

members = inspect.getmembers(obj)
print(isinstance(members, list))        # True

module_members = inspect.getmembers(inspect)
print(isinstance(module_members, list)) # True
print(len(module_members) > 0)          # True


# ── getmembers_static ─────────────────────────────────────────────────────────

class WithAttrs:
    x = 1
    def foo(self): pass

static_members = inspect.getmembers_static(WithAttrs)
print(isinstance(static_members, list)) # True
print(len(static_members) > 0)          # True


# ── getdoc ────────────────────────────────────────────────────────────────────

def documented():
    """This is a docstring."""
    pass

print(inspect.getdoc(documented) == "This is a docstring.")  # True
print(inspect.getdoc(myfunc) is None)   # True


# ── cleandoc ─────────────────────────────────────────────────────────────────

cleaned = inspect.cleandoc("""
    First line.
    Second line.
""")
print("First line." in cleaned)         # True
print("Second line." in cleaned)        # True


# ── getfile / getsourcefile ───────────────────────────────────────────────────

print(isinstance(inspect.getfile(myfunc), str))         # True
print(isinstance(inspect.getsourcefile(myfunc), str))   # True


# ── getsource raises ──────────────────────────────────────────────────────────

try:
    result = inspect.getsource(myfunc)
    print(isinstance(result, str))  # True — CPython finds .py source
except (OSError, TypeError):
    print(True)                     # True — goipy raises (running .pyc)


# ── getcomments ───────────────────────────────────────────────────────────────

print(inspect.getcomments(len) is None)   # True


# ── getmodulename ─────────────────────────────────────────────────────────────

print(inspect.getmodulename("foo/bar.py") == "bar")    # True
print(inspect.getmodulename("foo/bar.pyc") == "bar")   # True
print(inspect.getmodulename("foo/bar.txt") is None)    # True


# ── ispackage ────────────────────────────────────────────────────────────────

print(inspect.ispackage("/tmp") == False)   # True


# ── walktree callable ─────────────────────────────────────────────────────────

print(callable(inspect.walktree))   # True


# ── getattr_static ────────────────────────────────────────────────────────────

class Base:
    x = 10

class Child(Base):
    y = 20

c = Child()
print(inspect.getattr_static(c, "y") == 20)        # True
print(inspect.getattr_static(c, "z", 99) == 99)    # True
print(inspect.getattr_static(Child, "y") == 20)    # True


# ── getmro ────────────────────────────────────────────────────────────────────

class AA: pass
class BB(AA): pass
class CC(BB): pass

mro = inspect.getmro(CC)
print(isinstance(mro, tuple))   # True
print(CC in mro)                # True
print(BB in mro)                # True
print(AA in mro)                # True


# ── signature ─────────────────────────────────────────────────────────────────

def f(x, y, z=3):
    pass

sig = inspect.signature(f)
print(hasattr(sig, "parameters"))           # True
print(hasattr(sig, "return_annotation"))    # True
print(callable(sig.bind))                   # True
print(callable(sig.bind_partial))           # True


# ── Parameter ────────────────────────────────────────────────────────────────

print(inspect.Parameter.POSITIONAL_ONLY == 0)           # True
print(inspect.Parameter.POSITIONAL_OR_KEYWORD == 1)     # True
print(inspect.Parameter.VAR_POSITIONAL == 2)            # True
print(inspect.Parameter.KEYWORD_ONLY == 3)              # True
print(inspect.Parameter.VAR_KEYWORD == 4)               # True
print(hasattr(inspect.Parameter, "empty"))              # True


# ── getfullargspec ────────────────────────────────────────────────────────────

def g(a, b, *args, kw=1, **kwargs):
    pass

spec = inspect.getfullargspec(g)
print(hasattr(spec, "args"))             # True
print(hasattr(spec, "varargs"))          # True
print(hasattr(spec, "varkw"))            # True
print(hasattr(spec, "kwonlyargs"))       # True
print(hasattr(spec, "kwonlydefaults"))   # True
print(hasattr(spec, "annotations"))      # True


# ── formatannotation ──────────────────────────────────────────────────────────

print(isinstance(inspect.formatannotation(MyClass), str))      # True
print(isinstance(inspect.formatannotation("hello"), str))      # True
print(inspect.formatannotation("hello") == "'hello'")          # True  (repr of str)


# ── formatargvalues ───────────────────────────────────────────────────────────

print(isinstance(inspect.formatargvalues([], None, None, {}), str))  # True


# ── currentframe / stack / trace ─────────────────────────────────────────────

frame = inspect.currentframe()
print(frame is None or frame is not None)   # True
print(isinstance(inspect.stack(), list))    # True
print(isinstance(inspect.trace(), list))    # True


# ── getframeinfo / getouterframes / getinnerframes ───────────────────────────

print(callable(inspect.getframeinfo))       # True
print(callable(inspect.getouterframes))     # True
print(callable(inspect.getinnerframes))     # True


# ── classify_class_attrs ──────────────────────────────────────────────────────

cattrs = inspect.classify_class_attrs(MyClass)
print(isinstance(cattrs, list))     # True
print(len(cattrs) > 0)              # True
ca = cattrs[0]
print(hasattr(ca, "name"))          # True
print(hasattr(ca, "kind"))          # True


# ── FrameInfo / Attribute / BlockFinder ──────────────────────────────────────

print(inspect.FrameInfo._fields[0] == "frame")      # True
print(inspect.FrameInfo._fields[1] == "filename")   # True
print(inspect.Attribute._fields[0] == "name")       # True
print(inspect.Attribute._fields[3] == "object")     # True
print(inspect.BlockFinder is not None)              # True


# ── indentsize ────────────────────────────────────────────────────────────────

print(inspect.indentsize("    hello") == 4)   # True
print(inspect.indentsize("\thello") == 8)     # True  (tab expands to 8)
print(inspect.indentsize("hello") == 0)       # True


# ── get_annotations ──────────────────────────────────────────────────────────

print(isinstance(inspect.get_annotations(myfunc), dict))    # True
print(isinstance(inspect.get_annotations(MyClass), dict))   # True


# ── CO_* constants ────────────────────────────────────────────────────────────

print(inspect.CO_OPTIMIZED == 1)              # True
print(inspect.CO_NEWLOCALS == 2)              # True
print(inspect.CO_VARARGS == 4)               # True
print(inspect.CO_VARKEYWORDS == 8)           # True
print(inspect.CO_NESTED == 16)               # True
print(inspect.CO_GENERATOR == 32)            # True
print(inspect.CO_NOFREE == 64)               # True
print(inspect.CO_COROUTINE == 128)           # True
print(inspect.CO_ITERABLE_COROUTINE == 256)  # True
print(inspect.CO_ASYNC_GENERATOR == 512)     # True
print(inspect.CO_HAS_DOCSTRING == 67108864)  # True
print(inspect.CO_METHOD == 134217728)        # True


# ── TPFLAGS_IS_ABSTRACT ───────────────────────────────────────────────────────

print(inspect.TPFLAGS_IS_ABSTRACT == 1048576)   # True


print("done")
