import __main__

# Module identity
print(__main__.__name__ == '__main__')           # True

# __doc__ is None by default for a script
print(__main__.__doc__ is None)                  # True

# __annotations__ is a dict
print(isinstance(__main__.__annotations__, dict)) # True

# __spec__ is None when run as a script
print(__main__.__spec__ is None)                 # True

# __loader__ is None for scripts
print(__main__.__loader__ is None)               # True

# __package__ is None for top-level scripts
print(__main__.__package__ is None)              # True

# __builtins__ is present
print(hasattr(__main__, '__builtins__'))          # True

# The running script itself has __name__ == '__main__'
print(__name__ == '__main__')                    # True

# __main__ module is the live namespace — variables defined here are visible
x_sentinel = 42
print(__main__.x_sentinel == 42)                 # True

# The classic idiom works
flag = False
if __name__ == '__main__':
    flag = True
print(flag)                                      # True

# __main__.__annotations__ reflects script-level annotations
# (In CPython these accumulate as annotations are declared.)
# We only verify it starts as an empty dict or grows as a dict.
ann = __main__.__annotations__
print(isinstance(ann, dict))                     # True

# Re-importing returns the same module object (cached)
import __main__ as m2
print(m2.__name__ == '__main__')                 # True
print(m2 is __main__)                            # True — same cached module

# __main__ has the __builtins__ attribute pointing to the builtins namespace
import builtins
print(__main__.__builtins__ is builtins or isinstance(__main__.__builtins__, dict) or hasattr(__main__.__builtins__, '__name__'))  # True

print("done")
