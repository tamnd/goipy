"""Tests for ctypes module."""
import ctypes

# --- Simple types: instantiation and .value ---
print(ctypes.c_int(42).value == 42)           # True
print(ctypes.c_int(-1).value == -1)           # True
print(ctypes.c_int().value == 0)              # True
print(ctypes.c_double(3.14).value == 3.14)    # True
print(ctypes.c_bool(True).value == True)      # True
print(ctypes.c_bool(False).value == False)    # True
print(ctypes.c_char(b'A').value == b'A')      # True
print(ctypes.c_byte(127).value == 127)        # True
print(ctypes.c_ubyte(255).value == 255)       # True
print(ctypes.c_short(1000).value == 1000)     # True
print(ctypes.c_long(99).value == 99)          # True
print(ctypes.c_float(1.5).value == 1.5)       # True
print(ctypes.c_char_p(b"hello").value == b"hello")    # True
print(ctypes.c_char_p(None).value is None)            # True
print(ctypes.c_void_p(0).value == 0)                  # True
print(ctypes.c_wchar_p("hi").value == "hi")           # True

# --- sizeof ---
print(ctypes.sizeof(ctypes.c_int) == 4)       # True
print(ctypes.sizeof(ctypes.c_double) == 8)    # True
print(ctypes.sizeof(ctypes.c_bool) == 1)      # True
print(ctypes.sizeof(ctypes.c_char) == 1)      # True
print(ctypes.sizeof(ctypes.c_short) == 2)     # True
print(ctypes.sizeof(ctypes.c_long) == 8)      # True
print(ctypes.sizeof(ctypes.c_float) == 4)     # True
print(ctypes.sizeof(ctypes.c_longlong) == 8)  # True
# sizeof on instance
print(ctypes.sizeof(ctypes.c_int(0)) == 4)    # True

# --- Structure ---
class Point(ctypes.Structure):
    _fields_ = [('x', ctypes.c_int), ('y', ctypes.c_int)]

p = Point(1, 2)
print(p.x == 1)    # True
print(p.y == 2)    # True
print(ctypes.sizeof(Point) == 8)   # True (2 x c_int = 4+4)

class Rect(ctypes.Structure):
    _fields_ = [('left', ctypes.c_int), ('top', ctypes.c_int),
                ('right', ctypes.c_int), ('bottom', ctypes.c_int)]
r = Rect(0, 0, 100, 200)
print(r.right == 100)   # True
print(ctypes.sizeof(Rect) == 16)  # True

# --- create_string_buffer ---
buf = ctypes.create_string_buffer(10)
print(len(buf) == 10)     # True
buf2 = ctypes.create_string_buffer(b"hello")
print(buf2.value == b"hello")   # True

# --- CDLL (stub) ---
lib = ctypes.CDLL(None)
print(isinstance(lib, ctypes.CDLL))   # True

# --- POINTER / pointer ---
IntPtr = ctypes.POINTER(ctypes.c_int)
print(IntPtr.__name__ == 'LP_c_int')  # True
x = ctypes.c_int(7)
px = ctypes.pointer(x)
print(px.contents.value == 7)   # True

# --- byref ---
v = ctypes.c_int(5)
ref = ctypes.byref(v)
print(ref is not None)   # True

# --- addressof ---
addr = ctypes.addressof(ctypes.c_int(0))
print(isinstance(addr, int))   # True

# --- cast ---
v2 = ctypes.cast(ctypes.c_int(10), ctypes.c_long)
print(isinstance(v2, ctypes.c_long))   # True

# --- get_errno / set_errno ---
ctypes.set_errno(42)
print(ctypes.get_errno() == 42)   # True

# --- constants ---
print(ctypes.RTLD_LOCAL == 0)    # True
print(ctypes.RTLD_GLOBAL == 256)  # True

# --- Union ---
class IntOrBytes(ctypes.Union):
    _fields_ = [('i', ctypes.c_int), ('b', ctypes.c_byte)]
print(ctypes.sizeof(IntOrBytes) == 4)  # True (max of c_int=4, c_byte=1)

# --- c_longdouble, c_size_t ---
print(ctypes.sizeof(ctypes.c_longdouble) == 16)  # True
print(ctypes.sizeof(ctypes.c_size_t) == 8)       # True

# --- string_at stub ---
print(ctypes.string_at(0) == b'')   # True

# --- create_unicode_buffer ---
ubuf = ctypes.create_unicode_buffer(5)
print(len(ubuf) == 5)    # True
