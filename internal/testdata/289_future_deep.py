"""Comprehensive __future__ test — covers all public API from the Python docs."""
import __future__


# ── _Feature class ────────────────────────────────────────────────────────────

print(hasattr(__future__, '_Feature'))           # True
print(isinstance(__future__._Feature, type))     # True


# ── all_feature_names ─────────────────────────────────────────────────────────

names = __future__.all_feature_names
print(isinstance(names, list))                   # True
print(len(names) == 10)                          # True
print('division' in names)                        # True
print('annotations' in names)                    # True
print('generators' in names)                     # True

# order matches Python's definition
expected_names = [
    'nested_scopes', 'generators', 'division', 'absolute_import',
    'with_statement', 'print_function', 'unicode_literals',
    'barry_as_FLUFL', 'generator_stop', 'annotations',
]
print(names == expected_names)                   # True


# ── feature instances are _Feature ───────────────────────────────────────────

for name in __future__.all_feature_names:
    f = getattr(__future__, name)
    assert isinstance(f, __future__._Feature), f"Not a _Feature: {name}"
print("all_features_are_Feature ok")


# ── compiler_flag ─────────────────────────────────────────────────────────────

print(isinstance(__future__.division.compiler_flag, int))          # True
print(__future__.division.compiler_flag == 131072)                 # True
print(__future__.annotations.compiler_flag == 16777216)            # True
print(__future__.print_function.compiler_flag == 1048576)          # True
print(__future__.unicode_literals.compiler_flag == 2097152)        # True
print(__future__.barry_as_FLUFL.compiler_flag == 4194304)          # True
print(__future__.generator_stop.compiler_flag == 8388608)          # True
print(__future__.absolute_import.compiler_flag == 262144)          # True
print(__future__.with_statement.compiler_flag == 524288)           # True
print(__future__.generators.compiler_flag == 0)                    # True
print(__future__.nested_scopes.compiler_flag == 16)                # True


# ── getOptionalRelease ────────────────────────────────────────────────────────

opt = __future__.division.getOptionalRelease()
print(isinstance(opt, tuple))    # True
print(len(opt) == 5)             # True
print(opt[0] == 2)               # True  (major = 2)
print(opt[1] == 2)               # True  (minor = 2)
print(opt[3] == 'alpha')         # True

opt_ann = __future__.annotations.getOptionalRelease()
print(opt_ann == (3, 7, 0, 'beta', 1))   # True


# ── getMandatoryRelease ───────────────────────────────────────────────────────

mand = __future__.division.getMandatoryRelease()
print(isinstance(mand, tuple))   # True
print(mand == (3, 0, 0, 'alpha', 0))   # True

# annotations has no mandatory release yet → None
ann_mand = __future__.annotations.getMandatoryRelease()
print(ann_mand is None)          # True


# ── CO_FUTURE_* constants ─────────────────────────────────────────────────────

print(__future__.CO_FUTURE_DIVISION == 131072)          # True
print(__future__.CO_FUTURE_ABSOLUTE_IMPORT == 262144)   # True
print(__future__.CO_FUTURE_WITH_STATEMENT == 524288)    # True
print(__future__.CO_FUTURE_PRINT_FUNCTION == 1048576)   # True
print(__future__.CO_FUTURE_UNICODE_LITERALS == 2097152) # True
print(__future__.CO_FUTURE_BARRY_AS_BDFL == 4194304)    # True
print(__future__.CO_FUTURE_GENERATOR_STOP == 8388608)   # True
print(__future__.CO_FUTURE_ANNOTATIONS == 16777216)     # True
print(__future__.CO_GENERATOR_ALLOWED == 0)             # True
print(__future__.CO_NESTED == 16)                       # True


# ── _Feature created directly ─────────────────────────────────────────────────

f = __future__._Feature((3, 0, 0, 'final', 0), (4, 0, 0, 'alpha', 0), 42)
print(isinstance(f, __future__._Feature))     # True
print(f.compiler_flag == 42)                  # True
print(f.getOptionalRelease() == (3, 0, 0, 'final', 0))   # True
print(f.getMandatoryRelease() == (4, 0, 0, 'alpha', 0))  # True

# mandatory = None case
f2 = __future__._Feature((3, 0, 0, 'alpha', 0), None, 0)
print(f2.getMandatoryRelease() is None)       # True


print("done")
