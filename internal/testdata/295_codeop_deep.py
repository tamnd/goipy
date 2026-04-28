"""Comprehensive codeop module test — covers all public API from the Python docs."""
import codeop


# ── Constants ─────────────────────────────────────────────────────────────────

print(codeop.PyCF_DONT_IMPLY_DEDENT == 512)            # True
print(codeop.PyCF_ALLOW_INCOMPLETE_INPUT == 16384)     # True
print(codeop.PyCF_ONLY_AST == 1024)                    # True


# ── compile_command ───────────────────────────────────────────────────────────

r1 = codeop.compile_command("x = 1")
print(r1 is not None)                                  # True — complete

r2 = codeop.compile_command("if True:")
print(r2 is None)                                      # True — incomplete

r3 = codeop.compile_command("for i in range(10):")
print(r3 is None)                                      # True — incomplete

r4 = codeop.compile_command("def foo():")
print(r4 is None)                                      # True — incomplete

r5 = codeop.compile_command("print('hello')")
print(r5 is not None)                                  # True — complete

r6 = codeop.compile_command("")
print(r6 is not None)                                  # True — empty is complete

r7 = codeop.compile_command("x = (1 +")
print(r7 is None)                                      # True — unbalanced paren

r8 = codeop.compile_command("x = 1", filename="test.py")
print(r8 is not None)                                  # True

r9 = codeop.compile_command("x = 1", symbol="single")
print(r9 is not None)                                  # True

r10 = codeop.compile_command("x = 1", flags=codeop.PyCF_DONT_IMPLY_DEDENT)
print(r10 is not None)                                 # True — flags respected

r11 = codeop.compile_command("class Bar:")
print(r11 is None)                                     # True — incomplete


# ── Compile class ─────────────────────────────────────────────────────────────

c = codeop.Compile()
print(type(c).__name__ == "Compile")                   # True
print(callable(c))                                     # True

# initial flags = PyCF_DONT_IMPLY_DEDENT | PyCF_ALLOW_INCOMPLETE_INPUT
print(c.flags == (codeop.PyCF_DONT_IMPLY_DEDENT | codeop.PyCF_ALLOW_INCOMPLETE_INPUT))  # True
print(c.flags == 16896)                                # True

# calling with complete source returns a code object
r12 = c("x = 1", "<input>", "single")
print(r12 is not None)                                 # True

# calling with incomplete source raises an exception
try:
    c("if True:", "<input>", "single")
    print(False)                                       # should not reach here
except Exception:
    print(True)                                        # True — raises on incomplete

# two Compile instances are independent
c2 = codeop.Compile()
print(c2.flags == 16896)                               # True


# ── CommandCompiler class ─────────────────────────────────────────────────────

cc = codeop.CommandCompiler()
print(type(cc).__name__ == "CommandCompiler")          # True
print(callable(cc))                                    # True

# has a compiler attribute that is a Compile instance
print(isinstance(cc.compiler, codeop.Compile))        # True

# complete source → code object
r13 = cc("x = 1")
print(r13 is not None)                                 # True

# incomplete source → None
r14 = cc("if True:")
print(r14 is None)                                     # True

r15 = cc("def bar():")
print(r15 is None)                                     # True

r16 = cc("class Baz:")
print(r16 is None)                                     # True

r17 = cc("x = (1 +")
print(r17 is None)                                     # True

# with keyword args
r18 = cc("x = 1", filename="my.py")
print(r18 is not None)                                 # True

r19 = cc("x = 1", symbol="single")
print(r19 is not None)                                 # True


print("done")
