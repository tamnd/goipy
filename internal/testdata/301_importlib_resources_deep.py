import importlib.resources as ir
import importlib.resources.abc as irabc

# ── api surface ───────────────────────────────────────────────────────────────
print(callable(ir.files))
print(callable(ir.as_file))
print(callable(ir.read_binary))
print(callable(ir.read_text))
print(callable(ir.open_binary))
print(callable(ir.open_text))
print(callable(ir.is_resource))
print(callable(ir.contents))
print(callable(ir.path))

# ── files() with a real package ───────────────────────────────────────────────
import testpkgres

f = ir.files(testpkgres)
print(f.is_dir())
print(f.name)
print(f.is_file())

# joinpath to a text resource
r = f.joinpath('greeting.txt')
print(r.is_file())
print(r.is_dir())
print(r.name)

text = r.read_text(encoding='utf-8')
print(text.strip())

data = r.read_bytes()
print(type(data).__name__)
print(len(data) > 0)

# open() on resource file
with r.open('r') as fp:
    line = fp.readline()
    print(line.strip())

# iterdir lists the package directory
items = list(f.iterdir())
names = sorted(item.name for item in items)
print(type(items).__name__)
print('greeting.txt' in names)
print('__init__.py' in names or '__init__.pyc' in names)

# files() with package string name
f2 = ir.files('testpkgres')
print(f2.is_dir())

# ── legacy API ────────────────────────────────────────────────────────────────
print(ir.is_resource(testpkgres, 'greeting.txt'))
print(ir.is_resource(testpkgres, '_no_such_file_xyz.bin'))

c = list(ir.contents(testpkgres))
print(type(c).__name__)
print('greeting.txt' in c)

data2 = ir.read_binary(testpkgres, 'greeting.txt')
print(type(data2).__name__)
print(len(data2) > 0)

text2 = ir.read_text(testpkgres, 'greeting.txt', encoding='utf-8')
print(text2.strip())

# open_binary
with ir.open_binary(testpkgres, 'greeting.txt') as fp:
    d = fp.read()
    print(type(d).__name__)
    print(len(d) > 0)

# open_text
with ir.open_text(testpkgres, 'greeting.txt') as fp:
    t = fp.read()
    print(t.strip())

# ── as_file / path context managers ──────────────────────────────────────────
r2 = ir.files(testpkgres).joinpath('greeting.txt')
with ir.as_file(r2) as p:
    print(type(p).__name__)

with ir.path(testpkgres, 'greeting.txt') as p:
    print(type(p).__name__)

# ── importlib.resources.abc ───────────────────────────────────────────────────
print(callable(irabc.Traversable))
print(callable(irabc.TraversableResources))
print(callable(irabc.ResourceReader))

print('done')
