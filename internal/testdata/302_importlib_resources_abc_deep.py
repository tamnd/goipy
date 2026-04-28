import importlib.resources.abc as irabc

# ── 1. API surface ─────────────────────────────────────────────────────────────
print(callable(irabc.Traversable))
print(callable(irabc.TraversableResources))
print(callable(irabc.ResourceReader))
print(callable(irabc.TraversalError))

# ── 2. TraversalError is a catchable exception ─────────────────────────────────
try:
    raise irabc.TraversalError("path traversal failed")
except Exception as e:
    print(type(e).__name__)
    print(str(e))

# ── 3. Inheritance chain ────────────────────────────────────────────────────────
print(issubclass(irabc.TraversableResources, irabc.ResourceReader))

# ── 4. Concrete Traversable subclass ───────────────────────────────────────────
import io

class MyTraversable(irabc.Traversable):
    def __init__(self, name, data=None, children=None):
        self._name = name
        self._data = data          # bytes if file, None if dir
        self._children = children or []

    @property
    def name(self):
        return self._name

    def is_dir(self):
        return self._data is None

    def is_file(self):
        return self._data is not None

    def iterdir(self):
        return iter(self._children)

    def open(self, mode='r', encoding='utf-8', **kwargs):
        if 'b' in mode:
            return io.BytesIO(self._data if self._data is not None else b'')
        text = (self._data or b'').decode(encoding)
        return io.StringIO(text)

    def joinpath(self, *descendants):
        if not descendants:
            return self
        name = descendants[0]
        for c in self._children:
            if c.name == name:
                child = c
                break
        else:
            child = MyTraversable(name)
        if len(descendants) == 1:
            return child
        return child.joinpath(*descendants[1:])

root = MyTraversable('root', None, [
    MyTraversable('hello.txt', b'Hello, ABC!'),
    MyTraversable('data.bin', b'\x00\x01\x02'),
])

print(root.name)
print(root.is_dir())
print(root.is_file())

names = sorted(c.name for c in root.iterdir())
print(names)

f = root.joinpath('hello.txt')
print(f.name)
print(f.is_file())

# read_bytes uses Traversable.read_bytes() default → self.open('rb').read()
data = f.read_bytes()
print(type(data).__name__)
print(data)

# read_text uses Traversable.read_text() default → self.open(encoding=...).read()
text = f.read_text(encoding='utf-8')
print(text)

# __truediv__ sugar for joinpath
f2 = root / 'data.bin'
print(f2.name)
data2 = f2.read_bytes()
print(type(data2).__name__)
print(len(data2))

# multi-level joinpath
nested_root = MyTraversable('pkg', None, [
    MyTraversable('sub', None, [
        MyTraversable('deep.txt', b'deep content'),
    ]),
])
deep = nested_root.joinpath('sub', 'deep.txt')
print(deep.name)
print(deep.read_text())

# ── 5. TraversableResources subclass ───────────────────────────────────────────
class MyTraversableResources(irabc.TraversableResources):
    def __init__(self, traversable):
        self._root = traversable

    def files(self):
        return self._root

r = MyTraversableResources(root)

# open_resource → self.files().joinpath(resource).open('rb')
with r.open_resource('hello.txt') as fp:
    content = fp.read()
    print(type(content).__name__)
    print(content)

# is_resource → self.files().joinpath(path).is_file()
print(r.is_resource('hello.txt'))
print(r.is_resource('nonexistent'))

# contents → names from iterdir
contents = sorted(r.contents())
print(type(contents).__name__)
print(contents)

# resource_path → always raises FileNotFoundError
try:
    r.resource_path('hello.txt')
except FileNotFoundError:
    print('FileNotFoundError')

print('done')
