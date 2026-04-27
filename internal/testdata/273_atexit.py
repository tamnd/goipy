import atexit

# register and count
def hello(name):
    print(f"bye {name}")

def greet():
    print("goodbye world")

atexit.register(hello, "alice")
atexit.register(greet)
print(atexit._ncallbacks())  # 2

# unregister by function identity
atexit.unregister(hello)
print(atexit._ncallbacks())  # 1

# register returns the function
def cleanup():
    print("cleanup done")

fn = atexit.register(cleanup)
print(fn is cleanup)  # True
print(atexit._ncallbacks())  # 2

# _run_exitfuncs calls in LIFO order
atexit._run_exitfuncs()  # cleanup done, then goodbye world

# _clear resets the list
atexit._clear()
print(atexit._ncallbacks())  # 0

# register with args and kwargs
def add(a, b, extra=0):
    print(a + b + extra)

atexit.register(add, 1, 2, extra=10)
atexit._run_exitfuncs()  # 13
atexit._clear()

print("done")
