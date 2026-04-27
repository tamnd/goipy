import inspect

# ismodule / isclass / isfunction / isbuiltin / ismethod
import os
print(inspect.ismodule(os))          # True
print(inspect.ismodule(42))          # False

class MyClass:
    def method(self):
        pass

print(inspect.isclass(MyClass))      # True
print(inspect.isclass(42))           # False

def myfunc():
    pass

print(inspect.isfunction(myfunc))    # True
print(inspect.isfunction(42))        # False

print(inspect.isbuiltin(len))        # True
print(inspect.isbuiltin(myfunc))     # False

obj = MyClass()
print(inspect.ismethod(obj.method))  # True
print(inspect.ismethod(myfunc))      # False

# isroutine
print(inspect.isroutine(myfunc))     # True
print(inspect.isroutine(len))        # True
print(inspect.isroutine(42))         # False

# isgenerator / isgeneratorfunction
def gen():
    yield 1

g = gen()
print(inspect.isgenerator(g))         # True
print(inspect.isgeneratorfunction(gen)) # True
print(inspect.isgenerator(42))         # False

# getmro
class A: pass
class B(A): pass
mro = inspect.getmro(B)
print(isinstance(mro, tuple))  # True
print(B in mro)                # True
print(A in mro)                # True

# getdoc returns None for functions without docstring
print(inspect.getdoc(myfunc) is None)  # True

# getfile
print(isinstance(inspect.getfile(myfunc), str))  # True

# getmodulename
print(inspect.getmodulename("foo/bar.py") == "bar")  # True
print(inspect.getmodulename("foo/bar.pyc") == "bar")  # True
print(inspect.getmodulename("foo/bar.txt") is None)  # True

# cleandoc
print(inspect.cleandoc("  hello  ") == "hello")  # True

# signature returns an object with parameters
sig = inspect.signature(myfunc)
print(hasattr(sig, "parameters"))  # True

# Parameter constants
print(inspect.Parameter.POSITIONAL_OR_KEYWORD == 1)  # True
print(inspect.Parameter.VAR_POSITIONAL == 2)         # True
print(inspect.Parameter.KEYWORD_ONLY == 3)           # True
print(inspect.Parameter.VAR_KEYWORD == 4)            # True

# getattr_static
class C:
    x = 10
c = C()
print(inspect.getattr_static(c, "x") == 10)   # True
print(inspect.getattr_static(c, "z", 99) == 99)  # True

print("done")
