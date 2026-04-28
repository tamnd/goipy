"""Comprehensive gc test — covers all public API from the Python docs."""
import gc


# ── enable / disable / isenabled ─────────────────────────────────────────────

gc.disable()
print(gc.isenabled())           # False
gc.enable()
print(gc.isenabled())           # True


# ── collect ───────────────────────────────────────────────────────────────────

n = gc.collect()
print(isinstance(n, int))       # True
print(n >= 0)                   # True
print(isinstance(gc.collect(0), int))   # True
print(isinstance(gc.collect(1), int))   # True
print(isinstance(gc.collect(2), int))   # True


# ── get_count ─────────────────────────────────────────────────────────────────

count = gc.get_count()
print(isinstance(count, tuple))     # True
print(len(count) == 3)              # True
print(isinstance(count[0], int))    # True
print(isinstance(count[1], int))    # True
print(isinstance(count[2], int))    # True


# ── get_threshold / set_threshold ────────────────────────────────────────────

t = gc.get_threshold()
print(isinstance(t, tuple))         # True
print(len(t) == 3)                  # True
print(t[0] == 2000)                 # True  (Python 3.14 default)
print(t[1] == 10)                   # True
print(isinstance(t[2], int))        # True  (0 in Python 3.14 incremental GC)
gc.set_threshold(100, 5)
t2 = gc.get_threshold()
print(t2[0] == 100)                 # True
print(t2[1] == 5)                   # True
gc.set_threshold(2000, 10)


# ── get_objects ───────────────────────────────────────────────────────────────

print(isinstance(gc.get_objects(), list))   # True


# ── get_stats ─────────────────────────────────────────────────────────────────

stats = gc.get_stats()
print(isinstance(stats, list))      # True
print(len(stats) == 3)              # True
s = stats[0]
print(isinstance(s, dict))          # True
print('collections' in s)           # True
print('collected' in s)             # True
print('uncollectable' in s)         # True


# ── set_debug / get_debug ─────────────────────────────────────────────────────

gc.set_debug(gc.DEBUG_STATS)
print(gc.get_debug() == gc.DEBUG_STATS)     # True
gc.set_debug(0)
print(gc.get_debug() == 0)                  # True


# ── DEBUG_* constants ─────────────────────────────────────────────────────────

print(gc.DEBUG_STATS == 1)          # True
print(gc.DEBUG_COLLECTABLE == 2)    # True
print(gc.DEBUG_UNCOLLECTABLE == 4)  # True
print(gc.DEBUG_SAVEALL == 32)       # True
print(gc.DEBUG_LEAK == 38)          # True


# ── is_tracked ────────────────────────────────────────────────────────────────

print(isinstance(gc.is_tracked([]), bool))  # True
print(isinstance(gc.is_tracked({}), bool))  # True
print(isinstance(gc.is_tracked(1), bool))   # True


# ── is_finalized ──────────────────────────────────────────────────────────────

class Foo:
    pass

print(gc.is_finalized(Foo()) == False)      # True


# ── freeze / unfreeze / get_freeze_count ─────────────────────────────────────

gc.freeze()
fc = gc.get_freeze_count()
print(isinstance(fc, int))      # True
print(fc >= 0)                  # True
gc.unfreeze()


# ── garbage ───────────────────────────────────────────────────────────────────

print(isinstance(gc.garbage, list))     # True


# ── callbacks ─────────────────────────────────────────────────────────────────

print(isinstance(gc.callbacks, list))   # True


# ── get_referrers / get_referents ─────────────────────────────────────────────

print(isinstance(gc.get_referrers(), list))         # True
print(isinstance(gc.get_referents(1, 2, 3), list))  # True


print("done")
