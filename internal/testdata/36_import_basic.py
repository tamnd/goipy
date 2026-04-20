import _36_mymod as mymod
from _36_mymod import add, greet, Counter

# plain module access
print(mymod.GREETING)
print(mymod.add(2, 3))
print(mymod.greet("world"))

# from-import binds names directly
print(add(10, 20))
print(greet("goipy"))

c = Counter(5)
print(c.bump())
print(c.bump(10))
print(c.n)

# module identity: repeated import returns the same object
import _36_mymod as again
print(mymod is again)
print(mymod._loaded_marker)
