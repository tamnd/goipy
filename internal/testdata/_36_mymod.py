# Helper module imported by 36_import_basic.py.
# Exercises module-level code: constants, functions, and a small class.

GREETING = "hello"

def add(x, y):
    return x + y

def greet(name):
    return f"{GREETING}, {name}"

class Counter:
    def __init__(self, start=0):
        self.n = start
    def bump(self, by=1):
        self.n += by
        return self.n

# Side effect that runs once at import time.
_loaded_marker = [1, 2, 3]
