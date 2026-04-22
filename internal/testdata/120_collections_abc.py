from collections.abc import (
    Hashable, Callable, Iterable, Iterator, Reversible, Generator,
    Sized, Container, Collection,
    Sequence, MutableSequence,
    Set, MutableSet,
    Mapping, MutableMapping,
    MappingView, KeysView, ItemsView, ValuesView,
    Awaitable, Coroutine, AsyncIterable, AsyncIterator, AsyncGenerator,
    Buffer,
)

# ===== Hashable =====
print(isinstance(1, Hashable))           # True
print(isinstance(1.5, Hashable))         # True
print(isinstance("s", Hashable))         # True
print(isinstance(b"b", Hashable))        # True
print(isinstance((1,2), Hashable))       # True
print(isinstance(frozenset(), Hashable)) # True
print(isinstance(None, Hashable))        # True
print(isinstance([], Hashable))          # False
print(isinstance({}, Hashable))          # False
print(isinstance(set(), Hashable))       # False

# ===== Callable =====
print(isinstance(len, Callable))         # True
print(isinstance(lambda: 0, Callable))   # True
print(isinstance(int, Callable))         # True
print(isinstance(1, Callable))           # False
print(isinstance("s", Callable))         # False

# ===== Iterable =====
print(isinstance([], Iterable))          # True
print(isinstance((), Iterable))          # True
print(isinstance("s", Iterable))         # True
print(isinstance({}, Iterable))          # True
print(isinstance(set(), Iterable))       # True
print(isinstance(frozenset(), Iterable)) # True
print(isinstance(b"b", Iterable))        # True
print(isinstance(bytearray(), Iterable)) # True
print(isinstance(range(5), Iterable))    # True
print(isinstance(iter([]), Iterable))    # True
print(isinstance(1, Iterable))           # False
print(isinstance(None, Iterable))        # False

# ===== Iterator =====
print(isinstance(iter([]), Iterator))    # True
print(isinstance([], Iterator))          # False
print(isinstance("s", Iterator))         # False

# generator is also an Iterator
def gen():
    yield 1
g = gen()
print(isinstance(g, Iterator))           # True
print(isinstance(g, Generator))          # True

# ===== Reversible =====
print(isinstance([], Reversible))        # True
print(isinstance((), Reversible))        # True
print(isinstance("s", Reversible))       # True
print(isinstance(b"b", Reversible))      # True
print(isinstance(bytearray(), Reversible)) # True
print(isinstance(range(5), Reversible))  # True
print(isinstance({}, Reversible))        # False
print(isinstance(set(), Reversible))     # False

# ===== Sized =====
print(isinstance([], Sized))             # True
print(isinstance((), Sized))             # True
print(isinstance("s", Sized))            # True
print(isinstance({}, Sized))             # True
print(isinstance(set(), Sized))          # True
print(isinstance(frozenset(), Sized))    # True
print(isinstance(b"b", Sized))           # True
print(isinstance(bytearray(), Sized))    # True
print(isinstance(range(5), Sized))       # True
print(isinstance(1, Sized))              # False

# ===== Container =====
print(isinstance([], Container))         # True
print(isinstance((), Container))         # True
print(isinstance("s", Container))        # True
print(isinstance({}, Container))         # True
print(isinstance(set(), Container))      # True
print(isinstance(frozenset(), Container))# True
print(isinstance(b"b", Container))       # True
print(isinstance(bytearray(), Container))# True
print(isinstance(1, Container))          # False

# ===== Collection =====
print(isinstance([], Collection))        # True
print(isinstance((), Collection))        # True
print(isinstance("s", Collection))       # True
print(isinstance({}, Collection))        # True
print(isinstance(set(), Collection))     # True
print(isinstance(1, Collection))         # False

# ===== Sequence =====
print(isinstance([], Sequence))          # True
print(isinstance((), Sequence))          # True
print(isinstance("s", Sequence))         # True
print(isinstance(b"b", Sequence))        # True
print(isinstance(bytearray(), Sequence)) # True
print(isinstance(range(5), Sequence))    # True
print(isinstance({}, Sequence))          # False
print(isinstance(set(), Sequence))       # False

# ===== MutableSequence =====
print(isinstance([], MutableSequence))   # True
print(isinstance(bytearray(), MutableSequence)) # True
print(isinstance((), MutableSequence))   # False
print(isinstance("s", MutableSequence))  # False

# ===== Set =====
print(isinstance(set(), Set))            # True
print(isinstance(frozenset(), Set))      # True
print(isinstance([], Set))               # False
print(isinstance({}, Set))               # False

# ===== MutableSet =====
print(isinstance(set(), MutableSet))     # True
print(isinstance(frozenset(), MutableSet)) # False
print(isinstance([], MutableSet))        # False

# ===== Mapping =====
print(isinstance({}, Mapping))           # True
print(isinstance([], Mapping))           # False
print(isinstance((), Mapping))           # False

# ===== MutableMapping =====
print(isinstance({}, MutableMapping))    # True
print(isinstance([], MutableMapping))    # False

# ===== collections types =====
from collections import deque, Counter, defaultdict, OrderedDict, ChainMap

print(isinstance(deque(), Iterable))     # True
print(isinstance(deque(), Sized))        # True
print(isinstance(deque(), Container))    # True
print(isinstance(deque(), Reversible))   # True

print(isinstance(Counter(), Mapping))    # True
print(isinstance(Counter(), MutableMapping)) # True
print(isinstance(defaultdict(None), Mapping)) # True
print(isinstance(OrderedDict(), Mapping))     # True
print(isinstance(OrderedDict(), MutableMapping)) # True

# ===== User class with required methods =====
class MyIterable:
    def __iter__(self):
        return iter([])

mi = MyIterable()
print(isinstance(mi, Iterable))          # True
print(isinstance(mi, Sized))             # False (no __len__)

class MyMapping:
    def __getitem__(self, key): return key
    def __iter__(self): return iter([])
    def __len__(self): return 0

mm = MyMapping()
print(isinstance(mm, Mapping))           # True
print(isinstance(mm, MutableMapping))    # False (no __setitem__/__delitem__)

class MyMutableMapping:
    def __getitem__(self, key): return key
    def __setitem__(self, key, val): pass
    def __delitem__(self, key): pass
    def __iter__(self): return iter([])
    def __len__(self): return 0

mmm = MyMutableMapping()
print(isinstance(mmm, Mapping))          # True
print(isinstance(mmm, MutableMapping))   # True

class MySeq:
    def __getitem__(self, idx): return idx
    def __len__(self): return 0

ms = MySeq()
print(isinstance(ms, Sequence))          # True

class MyCallable:
    def __call__(self): pass

mc = MyCallable()
print(isinstance(mc, Callable))          # True

class MySized:
    def __len__(self): return 5

mz = MySized()
print(isinstance(mz, Sized))             # True
print(isinstance(mz, Iterable))          # False (no __iter__)

# ===== register() =====
class MyVirtualSeq:
    pass

Sequence.register(MyVirtualSeq)
mvs = MyVirtualSeq()
print(isinstance(mvs, Sequence))         # True

# ===== Async ABCs =====
print(isinstance(None, Awaitable))       # False
print(isinstance(None, AsyncIterable))   # False

class MyAwaitable:
    def __await__(self): return iter([])

ma = MyAwaitable()
print(isinstance(ma, Awaitable))         # True

# ===== Buffer =====
print(isinstance(b"hi", Buffer))         # True
print(isinstance(bytearray(b"hi"), Buffer)) # True
print(isinstance("hi", Buffer))          # False
print(isinstance(1, Buffer))             # False

print('done')
