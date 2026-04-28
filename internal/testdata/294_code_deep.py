"""Comprehensive code module test — covers all public API from the Python docs."""
import code


# ── compile_command ───────────────────────────────────────────────────────────

r1 = code.compile_command("x = 1")
print(r1 is not None)                   # True — complete statement

r2 = code.compile_command("if True:")
print(r2 is None)                       # True — incomplete (needs body)

r3 = code.compile_command("for i in range(10):")
print(r3 is None)                       # True — incomplete

r4 = code.compile_command("def foo():")
print(r4 is None)                       # True — incomplete

r5 = code.compile_command("print('hello')")
print(r5 is not None)                   # True — complete

r6 = code.compile_command("")
print(r6 is not None)                   # True — empty is complete

r7 = code.compile_command("x = (1 +")
print(r7 is None)                       # True — unbalanced paren

r8 = code.compile_command("x = 1", filename="test.py")
print(r8 is not None)                   # True — complete with filename


# ── CommandCompiler ───────────────────────────────────────────────────────────

cc = code.CommandCompiler()
print(callable(cc))                     # True

r9 = cc("x = 1")
print(r9 is not None)                   # True — complete

r10 = cc("if True:")
print(r10 is None)                      # True — incomplete

r11 = cc("class Foo:")
print(r11 is None)                      # True — incomplete


# ── Quitter ───────────────────────────────────────────────────────────────────

q = code.Quitter("quit")
print(type(q).__name__ == "Quitter")    # True
print(q.name == "quit")                 # True
print("quit" in repr(q))               # True
print(callable(q))                      # True

try:
    q()
except SystemExit:
    print(True)                         # True — raises SystemExit


# ── InteractiveInterpreter ────────────────────────────────────────────────────

ii = code.InteractiveInterpreter()
print(isinstance(ii.locals, dict))      # True
print(isinstance(ii.compile, code.CommandCompiler))  # True

# runsource: complete
print(ii.runsource("x = 1") == False)  # True

# runsource: incomplete
print(ii.runsource("if True:") == True)  # True

# runsource: complete multi
print(ii.runsource("y = 2 + 2") == False)  # True

# runcode is callable
print(callable(ii.runcode))             # True

# showsyntaxerror / showtraceback callable
print(callable(ii.showsyntaxerror))     # True
print(callable(ii.showtraceback))       # True

# write is callable (writes to stderr)
print(callable(ii.write))              # True
ii.write("test")                        # no-op, no output captured here

# with custom locals
ii2 = code.InteractiveInterpreter({"x": 42})
print(ii2.locals.get("x") == 42)       # True


# ── InteractiveConsole ────────────────────────────────────────────────────────

ic = code.InteractiveConsole({"a": 1})
print(isinstance(ic, code.InteractiveInterpreter))  # True
print(ic.filename == "<console>")       # True
print(ic.local_exit == False)          # True
print(isinstance(ic.buffer, list))     # True
print(len(ic.buffer) == 0)             # True

# custom filename
ic2 = code.InteractiveConsole(filename="myfile.py")
print(ic2.filename == "myfile.py")      # True

# push: incomplete
result = ic.push("if True:")
print(result == True)                   # True — needs more
print(len(ic.buffer) > 0)              # True — buffer has line

# push: complete (clears buffer)
ic3 = code.InteractiveConsole()
result2 = ic3.push("x = 1")
print(result2 == False)                 # True — complete
print(len(ic3.buffer) == 0)            # True — buffer cleared

# resetbuffer
ic.resetbuffer()
print(len(ic.buffer) == 0)             # True

# raw_input returns string (or raises EOFError in non-interactive context)
try:
    ri = ic.raw_input("")
    print(isinstance(ri, str))         # True (goipy returns "")
except EOFError:
    print(True)                        # True (CPython raises EOFError in non-tty)

# interact is callable
print(callable(ic.interact))           # True

# InteractiveConsole also has runsource/runcode/write
print(callable(ic.runsource))          # True
print(callable(ic.runcode))            # True
print(callable(ic.write))             # True


# ── interact module-level function ────────────────────────────────────────────

print(callable(code.interact))         # True


print("done")
