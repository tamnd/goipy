import importlib

# Absolute dotted import returns the innermost module (unlike `import a.b`,
# which returns `a`).
leaf = importlib.import_module("_39pkg.sub.leaf")
print(leaf.inc(10))
print(leaf.tag())
print(leaf.__name__)

# Top-level single module
state = importlib.import_module("_41_state")
print(state.VALUE)
print(state.read())

# reload() re-runs the module body against the same dict — mutations to
# module attributes are reset.
state.VALUE = "mutated"
print(state.read())
importlib.reload(state)
print(state.VALUE)
print(state.read())

# Relative form: package= supplies the base for leading dots.
sub = importlib.import_module(".sub", package="_39pkg")
print(sub.__name__)

# Built-in modules resolve too.
asy = importlib.import_module("asyncio")
print(type(asy).__name__)

# Missing module raises ImportError through the façade.
try:
    importlib.import_module("_41_not_a_thing")
except ImportError as e:
    print("missing:", e)
