import dis

# ── 1. Constants ─────────────────────────────────────────────────────────────
print(dis.HAVE_ARGUMENT == 43)
print(dis.EXTENDED_ARG == 69)
print(dis.cmp_op == ('<', '<=', '==', '!=', '>', '>='))

# ── 2. opmap spot checks ─────────────────────────────────────────────────────
print(dis.opmap['NOP'] == 27)
print(dis.opmap['CACHE'] == 0)
print(dis.opmap['LOAD_CONST'] == 82)
print(dis.opmap['RETURN_VALUE'] == 35)
print(dis.opmap['COMPARE_OP'] == 56)
print(isinstance(dis.opmap, dict))

# ── 3. opname spot checks ────────────────────────────────────────────────────
print(dis.opname[0] == 'CACHE')
print(dis.opname[27] == 'NOP')
print(dis.opname[82] == 'LOAD_CONST')
print(len(dis.opname) == 267)

# ── 4. opcode category lists ─────────────────────────────────────────────────
print(dis.hasconst == [82])
print(dis.hasjabs == [])
print(dis.hascompare == [56])
print(dis.hasfree == [62, 90, 97, 111])
print(isinstance(dis.hasarg, list))
print(isinstance(dis.hasname, list))
print(isinstance(dis.hasjump, list))
print(isinstance(dis.hasjrel, list))
print(isinstance(dis.haslocal, list))
print(isinstance(dis.hasexc, list))

# ── 5. Instruction._fields ───────────────────────────────────────────────────
print(dis.Instruction._fields == (
    'opname', 'opcode', 'arg', 'argval', 'argrepr',
    'offset', 'start_offset', 'starts_line', 'line_number',
    'label', 'positions', 'cache_info',
))

# ── 6. Positions._fields ─────────────────────────────────────────────────────
print(dis.Positions._fields == ('lineno', 'end_lineno', 'col_offset', 'end_col_offset'))

# ── 7. code_info returns str ─────────────────────────────────────────────────
def f(): pass
print(isinstance(dis.code_info(f), str))

# ── 8. show_code returns None ────────────────────────────────────────────────
import io
print(dis.show_code(f, file=io.StringIO()) is None)

# ── 9. dis() returns None ────────────────────────────────────────────────────
print(dis.dis(f, file=io.StringIO()) is None)

# ── 10. stack_effect returns int ─────────────────────────────────────────────
print(isinstance(dis.stack_effect(dis.opmap['NOP']), int))

# ── 11. get_instructions returns iterable ────────────────────────────────────
print(isinstance(list(dis.get_instructions(f)), list))

# ── 12. findlinestarts is callable ───────────────────────────────────────────
print(callable(dis.findlinestarts))

# ── 13. findlabels returns list (called with empty bytes) ────────────────────
print(isinstance(dis.findlabels(bytes(10)), list))

# ── 14. COMPILER_FLAG_NAMES ──────────────────────────────────────────────────
print(dis.COMPILER_FLAG_NAMES[1] == 'OPTIMIZED')
print(dis.COMPILER_FLAG_NAMES[32] == 'GENERATOR')
print(isinstance(dis.COMPILER_FLAG_NAMES, dict))

# ── 15. __all__ membership ───────────────────────────────────────────────────
print('dis' in dis.__all__)
print('Bytecode' in dis.__all__)
print('Instruction' in dis.__all__)
print('opmap' in dis.__all__)
print('opname' in dis.__all__)

# ── 16. Callables ────────────────────────────────────────────────────────────
print(callable(dis.dis))
print(callable(dis.disassemble))
print(callable(dis.distb))
print(callable(dis.disco))
print(callable(dis.code_info))
print(callable(dis.show_code))
print(callable(dis.get_instructions))
print(callable(dis.findlinestarts))
print(callable(dis.findlabels))
print(callable(dis.stack_effect))

print('done')
