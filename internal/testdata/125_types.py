import types

# ===== NoneType / EllipsisType / NotImplementedType =====
print(isinstance(None, types.NoneType))                   # True
print(isinstance(..., types.EllipsisType))                # True
print(isinstance(NotImplemented, types.NotImplementedType))  # True
print(types.NoneType)                                     # <class 'NoneType'>

# ===== FunctionType / LambdaType =====
def f(): pass
print(isinstance(f, types.FunctionType))                  # True
print(isinstance(lambda: None, types.LambdaType))         # True
print(types.FunctionType is types.LambdaType)             # True

# ===== BuiltinFunctionType / BuiltinMethodType =====
print(isinstance(len, types.BuiltinFunctionType))         # True
print(isinstance(len, types.BuiltinMethodType))           # True
print(types.BuiltinFunctionType is types.BuiltinMethodType)  # True

# ===== MethodType =====
class MyClass:
    def method(self): pass
obj = MyClass()
print(isinstance(obj.method, types.MethodType))           # True

# ===== GeneratorType =====
def gen():
    yield 1
    yield 2
g = gen()
print(isinstance(g, types.GeneratorType))                 # True
print(next(g))                                            # 1

# ===== ModuleType isinstance =====
print(isinstance(types, types.ModuleType))                # True

# ===== SimpleNamespace =====
ns = types.SimpleNamespace(x=1, y=2)
print(ns.x)                    # 1
print(ns.y)                    # 2
print(repr(ns))                # namespace(x=1, y=2)

ns.z = 99
print(ns.z)                    # 99

ns2 = types.SimpleNamespace(x=1, y=2)
ns3 = types.SimpleNamespace(x=1, y=2)
print(ns2 == ns3)              # True
print(ns2 == ns)               # False  (ns has extra z=99)

del ns2.x
try:
    _ = ns2.x
except AttributeError:
    print('AttributeError')    # AttributeError

# SimpleNamespace with no args
empty = types.SimpleNamespace()
print(repr(empty))             # namespace()

# ===== MappingProxyType =====
d = {'a': 1, 'b': 2}
mp = types.MappingProxyType(d)
print(mp['a'])                 # 1
print(mp['b'])                 # 2
print(len(mp))                 # 2
print('a' in mp)               # True
print('z' in mp)               # False
print(sorted(mp.keys()))       # ['a', 'b']
print(sorted(mp.values()))     # [1, 2]
print(sorted(mp.items()))      # [('a', 1), ('b', 2)]
print(mp.get('a'))             # 1
print(mp.get('z', 0))          # 0
print(mp.copy())               # {'a': 1, 'b': 2}
try:
    mp['a'] = 99
except TypeError:
    print('TypeError')         # TypeError

# ===== new_class =====
Animal = types.new_class('Animal')
print(Animal.__name__)         # Animal
a = Animal()
print(isinstance(a, Animal))  # True

def setup_dog(ns):
    ns['sound'] = 'woof'
    ns['speak'] = lambda self: self.sound

Dog = types.new_class('Dog', (), {}, setup_dog)
print(Dog.__name__)            # Dog
d2 = Dog()
print(d2.speak())              # woof

# ===== ModuleType constructor =====
m = types.ModuleType('mymod')
print(m.__name__)              # mymod
m.answer = 42
print(m.answer)                # 42

m2 = types.ModuleType('pkg', 'Package docstring')
print(m2.__name__)             # pkg
print(m2.__doc__)              # Package docstring

print('done')
