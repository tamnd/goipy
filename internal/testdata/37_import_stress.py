import _37_helper
from _37_helper import total, get, Box, RUN_COUNT

# Basic access through the module
print(_37_helper.total())
print(_37_helper.get(2))
print(_37_helper.get(99, "missing"))

# Names imported via `from` work as expected
print(total())
print(get(3))
b = Box(42)
print(b.unwrap())

# Re-importing the module does not re-execute its body
import _37_helper as again
print(_37_helper is again)
print(RUN_COUNT[0])  # exactly 1

# asyncio is still served by the built-in registry, not the filesystem
import asyncio
print(type(asyncio).__name__)

# ImportError: missing module
try:
    import _37_nope  # noqa
except ImportError as e:
    print("missing-module:", e)

# ImportError: missing name in an existing module
try:
    from _37_helper import not_a_thing  # noqa
except ImportError as e:
    # CPython appends "(from /abs/path/_37_helper.py)"; trim it off so the
    # fixture is path-independent.
    print("missing-name:", str(e).split(" (")[0])
