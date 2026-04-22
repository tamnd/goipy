"""Tests for new exception types: raise/catch, messages, sys.exit."""

import sys
import warnings

# --- SystemExit via sys.exit ---
try:
    sys.exit(0)
except SystemExit as e:
    print("SystemExit:", e.args[0])

try:
    sys.exit(42)
except SystemExit as e:
    print("SystemExit:", e.args[0])

try:
    sys.exit("error message")
except SystemExit as e:
    print("SystemExit:", e.args[0])

# --- GeneratorExit ---
try:
    raise GeneratorExit
except GeneratorExit:
    print("caught GeneratorExit")

# GeneratorExit is NOT caught by except Exception
try:
    raise GeneratorExit
except Exception:
    print("should not print")
except GeneratorExit:
    print("GeneratorExit not caught by Exception")

# --- KeyboardInterrupt ---
try:
    raise KeyboardInterrupt
except KeyboardInterrupt:
    print("caught KeyboardInterrupt")

try:
    raise KeyboardInterrupt
except Exception:
    print("should not print")
except KeyboardInterrupt:
    print("KeyboardInterrupt not caught by Exception")

# --- FloatingPointError ---
try:
    raise FloatingPointError("fp error")
except ArithmeticError as e:
    print("FloatingPointError as ArithmeticError:", str(e))

# --- BufferError ---
try:
    raise BufferError("buf err")
except Exception as e:
    print("BufferError:", str(e))

# --- MemoryError ---
try:
    raise MemoryError
except MemoryError:
    print("caught MemoryError")

# --- UnboundLocalError ---
try:
    raise UnboundLocalError("x before assignment")
except NameError as e:
    print("UnboundLocalError as NameError:", str(e))

# --- ModuleNotFoundError ---
try:
    import nonexistent_module_xyz
except ModuleNotFoundError as e:
    print("ModuleNotFoundError:", "nonexistent_module_xyz" in str(e))

try:
    import nonexistent_module_xyz
except ImportError as e:
    print("ModuleNotFoundError as ImportError: True")

# --- OSError subclasses ---
try:
    raise FileExistsError("already exists")
except OSError as e:
    print("FileExistsError as OSError:", str(e))

try:
    raise PermissionError("denied")
except OSError as e:
    print("PermissionError:", str(e))

try:
    raise TimeoutError("timed out")
except OSError as e:
    print("TimeoutError:", str(e))

try:
    raise IsADirectoryError("is a dir")
except OSError as e:
    print("IsADirectoryError:", str(e))

try:
    raise NotADirectoryError("not a dir")
except OSError as e:
    print("NotADirectoryError:", str(e))

try:
    raise InterruptedError("interrupted")
except OSError as e:
    print("InterruptedError:", str(e))

try:
    raise BlockingIOError("blocking")
except OSError as e:
    print("BlockingIOError:", str(e))

try:
    raise ChildProcessError("child")
except OSError as e:
    print("ChildProcessError:", str(e))

try:
    raise ProcessLookupError("no process")
except OSError as e:
    print("ProcessLookupError:", str(e))

# --- ConnectionError subclasses ---
try:
    raise BrokenPipeError("broken pipe")
except ConnectionError as e:
    print("BrokenPipeError as ConnectionError:", str(e))

try:
    raise ConnectionAbortedError("aborted")
except OSError as e:
    print("ConnectionAbortedError as OSError:", str(e))

try:
    raise ConnectionRefusedError("refused")
except ConnectionError as e:
    print("ConnectionRefusedError:", str(e))

try:
    raise ConnectionResetError("reset")
except OSError as e:
    print("ConnectionResetError as OSError:", str(e))

# --- SyntaxError family ---
try:
    raise SyntaxError("bad syntax")
except Exception as e:
    print("SyntaxError:", str(e))

try:
    raise IndentationError("bad indent")
except SyntaxError as e:
    print("IndentationError as SyntaxError:", str(e))

try:
    raise TabError("mixed tabs")
except IndentationError as e:
    print("TabError as IndentationError:", str(e))

# --- SystemError ---
try:
    raise SystemError("internal")
except Exception as e:
    print("SystemError:", str(e))

# --- ReferenceError ---
try:
    raise ReferenceError("dead weakref")
except Exception as e:
    print("ReferenceError:", str(e))

# --- UnicodeError family ---
try:
    raise UnicodeError("unicode error")
except ValueError as e:
    print("UnicodeError as ValueError:", str(e))

try:
    raise UnicodeDecodeError("utf-8", b"\xff", 0, 1, "invalid byte")
except UnicodeError as e:
    print("UnicodeDecodeError:", type(e).__name__)

try:
    raise UnicodeEncodeError("utf-8", "hello", 0, 1, "encode fail")
except UnicodeError as e:
    print("UnicodeEncodeError:", type(e).__name__)

try:
    raise UnicodeTranslateError("hello", 0, 1, "translate fail")
except UnicodeError as e:
    print("UnicodeTranslateError:", type(e).__name__)

# --- Warning hierarchy ---
try:
    raise DeprecationWarning("deprecated")
except Warning as e:
    print("DeprecationWarning as Warning:", str(e))

try:
    raise UserWarning("user warn")
except Exception as e:
    print("UserWarning as Exception:", str(e))

try:
    raise RuntimeWarning("runtime warn")
except Warning as e:
    print("RuntimeWarning:", str(e))

try:
    raise BytesWarning("bytes warn")
except Warning as e:
    print("BytesWarning:", str(e))

# --- warnings.warn ---
warnings.warn("test warning", UserWarning)
warnings.warn("deprecation notice", DeprecationWarning)

# --- EnvironmentError as OSError alias ---
print(issubclass(EnvironmentError, OSError))
