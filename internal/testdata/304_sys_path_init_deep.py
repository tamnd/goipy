import sys

# ── 1. sys.path ────────────────────────────────────────────────────────────────
print(isinstance(sys.path, list))
print(len(sys.path) >= 1)
print(isinstance(sys.path[0], str))

# mutation: append and remove
sys.path.append('/tmp/testdir_xyz')
print('/tmp/testdir_xyz' in sys.path)
sys.path.remove('/tmp/testdir_xyz')
print('/tmp/testdir_xyz' in sys.path)

# ── 2. sys.path_hooks ─────────────────────────────────────────────────────────
print(isinstance(sys.path_hooks, list))

# ── 3. sys.path_importer_cache ────────────────────────────────────────────────
print(isinstance(sys.path_importer_cache, dict))

# ── 4. sys.meta_path ─────────────────────────────────────────────────────────
print(isinstance(sys.meta_path, list))

# ── 5. Prefix attributes ──────────────────────────────────────────────────────
print(isinstance(sys.prefix, str))
print(isinstance(sys.exec_prefix, str))
print(isinstance(sys.base_prefix, str))
print(isinstance(sys.base_exec_prefix, str))

# ── 6. sys.platlibdir ─────────────────────────────────────────────────────────
print(isinstance(sys.platlibdir, str))
print(sys.platlibdir)

# ── 7. sys.stdlib_module_names ────────────────────────────────────────────────
print(isinstance(sys.stdlib_module_names, frozenset))
print('os' in sys.stdlib_module_names)
print('sys' in sys.stdlib_module_names)
print('json' in sys.stdlib_module_names)
print('asyncio' in sys.stdlib_module_names)
print('collections' not in sys.stdlib_module_names or 'collections' in sys.stdlib_module_names)
print(len(sys.stdlib_module_names) > 50)

# ── 8. sys.flags path-related attributes ──────────────────────────────────────
print(isinstance(sys.flags.ignore_environment, int))
print(isinstance(sys.flags.no_user_site, int))
print(isinstance(sys.flags.no_site, int))
# safe_path is 3.11+; just check it's accessible
_ = sys.flags.safe_path
print(True)

# ── 9. sys.abiflags ───────────────────────────────────────────────────────────
print(isinstance(sys.abiflags, str))

# ── 10. sys.float_repr_style ──────────────────────────────────────────────────
print(sys.float_repr_style in ('short', 'legacy'))

# ── 11. sys.hexversion ───────────────────────────────────────────────────────
print(isinstance(sys.hexversion, int))
print(sys.hexversion > 0)
# 3.14.x → 0x030e....
print(sys.hexversion >= 0x030e0000)

# ── 12. sys.int_info ─────────────────────────────────────────────────────────
ii = sys.int_info
print(isinstance(ii.bits_per_digit, int))
print(ii.bits_per_digit > 0)
print(isinstance(ii.sizeof_digit, int))
print(isinstance(ii.default_max_str_digits, int))
print(isinstance(ii.str_digits_check_threshold, int))

# ── 13. sys.float_info ───────────────────────────────────────────────────────
fi = sys.float_info
print(isinstance(fi.max, float))
print(fi.max > 1e300)
print(isinstance(fi.radix, int))
print(fi.radix == 2)
print(isinstance(fi.mant_dig, int))
print(fi.mant_dig == 53)
print(isinstance(fi.epsilon, float))
print(fi.epsilon < 1e-10)

# ── 14. sys.hash_info ────────────────────────────────────────────────────────
hi = sys.hash_info
print(isinstance(hi.width, int))
print(hi.width > 0)
print(isinstance(hi.modulus, int))
print(hi.modulus > 0)
print(isinstance(hi.algorithm, str))

# ── 15. sys.thread_info ──────────────────────────────────────────────────────
ti = sys.thread_info
print(isinstance(ti.name, str))
print(isinstance(ti.lock, str))

print('done')
