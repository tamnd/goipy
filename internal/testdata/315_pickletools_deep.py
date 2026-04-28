import pickletools
import pickle
import io

# ── 1. __all__ ────────────────────────────────────────────────────────────────
print(pickletools.__all__ == ['dis', 'genops', 'optimize'])

# ── 2. opcodes list length ────────────────────────────────────────────────────
print(len(pickletools.opcodes) == 68)
print(isinstance(pickletools.opcodes, list))

# ── 3. opcodes[0] = INT ───────────────────────────────────────────────────────
op0 = pickletools.opcodes[0]
print(op0.name == 'INT')
print(op0.code == 'I')
print(op0.proto == 0)
print(op0.arg is not None)
print(op0.arg.name == 'decimalnl_short')
print(isinstance(op0.stack_before, list))
print(isinstance(op0.stack_after, list))
print(op0.stack_after[0].name == 'int_or_bool')

# ── 4. STOP opcode ────────────────────────────────────────────────────────────
stop = None
for o in pickletools.opcodes:
    if o.name == 'STOP':
        stop = o
        break
print(stop is not None)
print(stop.code == '.')
print(stop.arg is None)
print(stop.proto == 0)

# ── 5. BININT1 opcode ────────────────────────────────────────────────────────
binint1 = None
for o in pickletools.opcodes:
    if o.name == 'BININT1':
        binint1 = o
        break
print(binint1 is not None)
print(binint1.code == 'K')
print(binint1.arg.name == 'uint1')

# ── 6. PROTO opcode ───────────────────────────────────────────────────────────
proto_op = None
for o in pickletools.opcodes:
    if o.name == 'PROTO':
        proto_op = o
        break
print(proto_op is not None)
print(proto_op.proto == 2)

# ── 7. Classes are types ──────────────────────────────────────────────────────
print(isinstance(pickletools.OpcodeInfo, type))
print(isinstance(pickletools.StackObject, type))
print(isinstance(pickletools.ArgumentDescriptor, type))

# ── 8. decode_long ───────────────────────────────────────────────────────────
print(pickletools.decode_long(b'') == 0)
print(pickletools.decode_long(b'\x00') == 0)
print(pickletools.decode_long(b'\x01') == 1)
print(pickletools.decode_long(b'\xff') == -1)
print(pickletools.decode_long(b'\xff\x00') == 255)
print(pickletools.decode_long(b'\x01\x00') == 1)

# ── 9. callable checks ────────────────────────────────────────────────────────
print(callable(pickletools.genops))
print(callable(pickletools.dis))
print(callable(pickletools.optimize))
print(callable(pickletools.decode_long))

# ── 10. optimize returns bytes ────────────────────────────────────────────────
p = pickle.dumps(42)
opt = pickletools.optimize(p)
print(isinstance(opt, bytes))

# ── 11. dis() returns None ────────────────────────────────────────────────────
print(pickletools.dis(p, out=io.StringIO()) is None)

# ── 12. genops returns iterable ───────────────────────────────────────────────
print(isinstance(list(pickletools.genops(p)), list))

# ── 13. reader stubs are callable ────────────────────────────────────────────
print(callable(pickletools.read_uint1))
print(callable(pickletools.read_uint2))
print(callable(pickletools.read_uint4))
print(callable(pickletools.read_uint8))
print(callable(pickletools.read_int4))
print(callable(pickletools.read_bytes1))
print(callable(pickletools.read_bytes4))
print(callable(pickletools.read_bytes8))
print(callable(pickletools.read_bytearray8))
print(callable(pickletools.read_string1))
print(callable(pickletools.read_string4))
print(callable(pickletools.read_stringnl))
print(callable(pickletools.read_stringnl_noescape))
print(callable(pickletools.read_stringnl_noescape_pair))
print(callable(pickletools.read_float8))
print(callable(pickletools.read_floatnl))
print(callable(pickletools.read_decimalnl_short))
print(callable(pickletools.read_decimalnl_long))
print(callable(pickletools.read_long1))
print(callable(pickletools.read_long4))
print(callable(pickletools.read_unicodestring1))
print(callable(pickletools.read_unicodestring4))
print(callable(pickletools.read_unicodestring8))
print(callable(pickletools.read_unicodestringnl))

# ── 14. stack_before/after are lists of StackObjects ─────────────────────────
append_op = None
for o in pickletools.opcodes:
    if o.name == 'APPEND':
        append_op = o
        break
print(append_op is not None)
print(len(append_op.stack_before) == 2)
print(append_op.stack_before[0].name == 'list')
print(append_op.stack_before[1].name == 'any')

print('done')
