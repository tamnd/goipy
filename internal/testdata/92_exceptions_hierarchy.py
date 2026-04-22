"""Tests for exception hierarchy and isinstance checks."""

# --- BaseException-level (not caught by except Exception) ---
print(issubclass(SystemExit, BaseException))
print(issubclass(SystemExit, Exception))
print(issubclass(KeyboardInterrupt, BaseException))
print(issubclass(KeyboardInterrupt, Exception))
print(issubclass(GeneratorExit, BaseException))
print(issubclass(GeneratorExit, Exception))

# --- Exception root ---
print(issubclass(Exception, BaseException))

# --- ArithmeticError family ---
print(issubclass(ArithmeticError, Exception))
print(issubclass(FloatingPointError, ArithmeticError))
print(issubclass(FloatingPointError, Exception))
print(issubclass(OverflowError, ArithmeticError))
print(issubclass(ZeroDivisionError, ArithmeticError))

# --- LookupError family ---
print(issubclass(LookupError, Exception))
print(issubclass(IndexError, LookupError))
print(issubclass(KeyError, LookupError))

# --- NameError family ---
print(issubclass(UnboundLocalError, NameError))
print(issubclass(UnboundLocalError, Exception))

# --- ImportError family ---
print(issubclass(ModuleNotFoundError, ImportError))
print(issubclass(ModuleNotFoundError, Exception))

# --- OSError family ---
print(issubclass(OSError, Exception))
print(issubclass(IOError, OSError))
print(issubclass(FileNotFoundError, OSError))
print(issubclass(FileExistsError, OSError))
print(issubclass(PermissionError, OSError))
print(issubclass(TimeoutError, OSError))
print(issubclass(IsADirectoryError, OSError))
print(issubclass(NotADirectoryError, OSError))
print(issubclass(InterruptedError, OSError))
print(issubclass(BlockingIOError, OSError))
print(issubclass(ChildProcessError, OSError))
print(issubclass(ProcessLookupError, OSError))

# --- ConnectionError family ---
print(issubclass(ConnectionError, OSError))
print(issubclass(BrokenPipeError, ConnectionError))
print(issubclass(ConnectionAbortedError, ConnectionError))
print(issubclass(ConnectionRefusedError, ConnectionError))
print(issubclass(ConnectionResetError, ConnectionError))

# --- RuntimeError family ---
print(issubclass(RuntimeError, Exception))
print(issubclass(NotImplementedError, RuntimeError))
print(issubclass(RecursionError, RuntimeError))

# --- SyntaxError family ---
print(issubclass(SyntaxError, Exception))
print(issubclass(IndentationError, SyntaxError))
print(issubclass(TabError, IndentationError))
print(issubclass(TabError, SyntaxError))

# --- ValueError / UnicodeError family ---
print(issubclass(ValueError, Exception))
print(issubclass(UnicodeError, ValueError))
print(issubclass(UnicodeDecodeError, UnicodeError))
print(issubclass(UnicodeEncodeError, UnicodeError))
print(issubclass(UnicodeTranslateError, UnicodeError))

# --- Warning family ---
print(issubclass(Warning, Exception))
print(issubclass(DeprecationWarning, Warning))
print(issubclass(PendingDeprecationWarning, Warning))
print(issubclass(RuntimeWarning, Warning))
print(issubclass(SyntaxWarning, Warning))
print(issubclass(UserWarning, Warning))
print(issubclass(FutureWarning, Warning))
print(issubclass(ImportWarning, Warning))
print(issubclass(UnicodeWarning, Warning))
print(issubclass(BytesWarning, Warning))
print(issubclass(ResourceWarning, Warning))
print(issubclass(EncodingWarning, Warning))

# --- Other new exception classes ---
print(issubclass(BufferError, Exception))
print(issubclass(MemoryError, Exception))
print(issubclass(ReferenceError, Exception))
print(issubclass(SystemError, Exception))
print(issubclass(StopAsyncIteration, Exception))
print(issubclass(EOFError, Exception))

# --- isinstance with instances ---
try:
    raise ValueError("test")
except ValueError as e:
    print(isinstance(e, ValueError))
    print(isinstance(e, Exception))
    print(isinstance(e, BaseException))
    print(isinstance(e, TypeError))

try:
    raise KeyError("k")
except LookupError as e:
    print(isinstance(e, KeyError))
    print(isinstance(e, LookupError))

try:
    raise FileNotFoundError("no file")
except OSError as e:
    print(isinstance(e, FileNotFoundError))
    print(isinstance(e, OSError))

# --- except tuple catches ---
try:
    raise OverflowError("big")
except (ValueError, ArithmeticError):
    print("caught arithmetic")

try:
    raise UnicodeDecodeError("utf-8", b"", 0, 1, "reason")
except (ValueError, UnicodeError) as e:
    print("caught unicode:", type(e).__name__)

# --- BaseExceptionGroup ---
print(issubclass(BaseExceptionGroup, BaseException))
print(issubclass(ExceptionGroup, Exception))
print(issubclass(ExceptionGroup, BaseExceptionGroup))
