import copyreg

# ===== module attributes =====
print(isinstance(copyreg.dispatch_table, dict))   # True
print(hasattr(copyreg, 'pickle'))                  # True
print(hasattr(copyreg, 'constructor'))             # True
print(hasattr(copyreg, 'add_extension'))           # True
print(hasattr(copyreg, 'remove_extension'))        # True
print(hasattr(copyreg, 'clear_extension_cache'))   # True

# ===== copyreg.constructor() =====
# callable succeeds
copyreg.constructor(int)
copyreg.constructor(list)
copyreg.constructor(lambda: None)
print('constructor ok')                            # constructor ok

# non-callable raises TypeError
try:
    copyreg.constructor(42)
except TypeError as e:
    print('TypeError for non-callable constructor') # TypeError for non-callable constructor

try:
    copyreg.constructor("hello")
except TypeError as e:
    print('TypeError for str constructor')          # TypeError for str constructor

# ===== copyreg.pickle() — register reducer =====
def my_reduce(obj):
    return (list, (list(obj),))

copyreg.pickle(tuple, my_reduce)
# Check it's stored in dispatch_table
print(tuple in copyreg.dispatch_table)             # True
print(copyreg.dispatch_table[tuple] is my_reduce)  # True

# Non-callable pickle_function raises TypeError
try:
    copyreg.pickle(tuple, 42)
except TypeError as e:
    print('TypeError for non-callable reducer')     # TypeError for non-callable reducer

# Non-callable constructor_ob raises TypeError
try:
    copyreg.pickle(tuple, my_reduce, 99)
except TypeError as e:
    print('TypeError for non-callable constructor_ob')  # TypeError for non-callable constructor_ob

# Valid constructor_ob
copyreg.pickle(tuple, my_reduce, list)
print('pickle with constructor ok')                # pickle with constructor ok

# Multiple types in dispatch_table
def int_reduce(obj):
    return (int, (int(obj),))

copyreg.pickle(int, int_reduce)
print(int in copyreg.dispatch_table)              # True
print(len(copyreg.dispatch_table) >= 2)           # True

# ===== add_extension / remove_extension =====
copyreg.add_extension('mymodule', 'myobj', 1)
print('add_extension ok')                          # add_extension ok

# Idempotent for same triple
copyreg.add_extension('mymodule', 'myobj', 1)
print('add_extension idempotent ok')               # add_extension idempotent ok

# Code 0 is out of range
try:
    copyreg.add_extension('m', 'n', 0)
except ValueError as e:
    print('ValueError code 0')                     # ValueError code 0

# Negative code out of range
try:
    copyreg.add_extension('m', 'n', -1)
except ValueError as e:
    print('ValueError negative code')              # ValueError negative code

# Different code for same (module, name) raises ValueError
try:
    copyreg.add_extension('mymodule', 'myobj', 2)
except ValueError as e:
    print('ValueError code conflict')              # ValueError code conflict

# Same code for different (module, name) raises ValueError
try:
    copyreg.add_extension('other', 'obj', 1)
except ValueError as e:
    print('ValueError name conflict')              # ValueError name conflict

# remove_extension
copyreg.remove_extension('mymodule', 'myobj', 1)
print('remove_extension ok')                       # remove_extension ok

# After removal, can add again
copyreg.add_extension('mymodule', 'myobj', 1)
print('re-add after remove ok')                   # re-add after remove ok

# remove non-existent raises ValueError
try:
    copyreg.remove_extension('no', 'such', 999)
except ValueError as e:
    print('ValueError remove non-existent')        # ValueError remove non-existent

# ===== clear_extension_cache =====
copyreg.clear_extension_cache()
print('clear_extension_cache ok')                  # clear_extension_cache ok

# After clear_cache, remove_extension still works (cache != registry)
copyreg.remove_extension('mymodule', 'myobj', 1)
print('remove after clear cache ok')               # remove after clear cache ok

# ===== multiple extensions =====
copyreg.add_extension('mod1', 'obj1', 10)
copyreg.add_extension('mod2', 'obj2', 20)
copyreg.add_extension('mod3', 'obj3', 30)
copyreg.remove_extension('mod2', 'obj2', 20)
# mod1 and mod3 still registered; mod2 gone
try:
    copyreg.remove_extension('mod2', 'obj2', 20)
except ValueError as e:
    print('mod2 correctly removed')               # mod2 correctly removed
copyreg.remove_extension('mod1', 'obj1', 10)
copyreg.remove_extension('mod3', 'obj3', 30)
print('multi extension ok')                       # multi extension ok

print('done')                                     # done
