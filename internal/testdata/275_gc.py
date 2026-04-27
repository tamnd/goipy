import gc

# enable/disable/isenabled
print(gc.isenabled())   # True
gc.disable()
print(gc.isenabled())   # False
gc.enable()
print(gc.isenabled())   # True

# collect returns int
n = gc.collect()
print(isinstance(n, int))  # True

# get_count returns a 3-tuple
count = gc.get_count()
print(len(count) == 3)  # True

# get_threshold / set_threshold
t = gc.get_threshold()
print(len(t) == 3)  # True
gc.set_threshold(500, 8, 8)
t2 = gc.get_threshold()
print(t2[0] == 500)  # True
print(t2[1] == 8)    # True

# get_objects returns a list
objs = gc.get_objects()
print(isinstance(objs, list))  # True

# is_tracked / is_finalized
print(gc.is_tracked(42))   # False
print(gc.is_finalized(42)) # False

# freeze / unfreeze / get_freeze_count
gc.freeze()
gc.unfreeze()
print(gc.get_freeze_count() == 0)  # True

# get_referents / get_referrers
print(isinstance(gc.get_referents(42), list))  # True
print(isinstance(gc.get_referrers(42), list))  # True

# debug flags
print(gc.DEBUG_SAVEALL == 32)  # True
print(gc.DEBUG_LEAK == 38)     # True

# callbacks list exists
print(isinstance(gc.callbacks, list))  # True

print("done")
