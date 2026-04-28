import symtable

# ── 1. Integer constants ───────────────────────────────────────────────────────
print(symtable.CELL == 5)
print(symtable.FREE == 4)
print(symtable.LOCAL == 1)
print(symtable.GLOBAL_EXPLICIT == 2)
print(symtable.GLOBAL_IMPLICIT == 3)
print(symtable.SCOPE_MASK == 15)
print(symtable.SCOPE_OFF == 12)
print(symtable.USE == 16)
print(symtable.DEF_GLOBAL == 1)
print(symtable.DEF_LOCAL == 2)
print(symtable.DEF_PARAM == 4)
print(symtable.DEF_NONLOCAL == 8)
print(symtable.DEF_IMPORT == 128)
print(symtable.DEF_BOUND == 134)
print(symtable.DEF_ANNOT == 256)
print(symtable.DEF_COMP_ITER == 512)
print(symtable.DEF_TYPE_PARAM == 1024)
print(symtable.DEF_COMP_CELL == 2048)

# ── 2. Classes exist ──────────────────────────────────────────────────────────
print(hasattr(symtable, 'SymbolTable'))
print(hasattr(symtable, 'Function'))
print(hasattr(symtable, 'Class'))
print(hasattr(symtable, 'Symbol'))

# ── 3. Class hierarchy ────────────────────────────────────────────────────────
print(issubclass(symtable.Function, symtable.SymbolTable))
print(issubclass(symtable.Class, symtable.SymbolTable))

# ── 4. symtable() returns SymbolTable ─────────────────────────────────────────
st = symtable.symtable('x = 1', '<string>', 'exec')
print(isinstance(st, symtable.SymbolTable))
print(st.get_type() == 'module')
print(st.get_name() == 'top')
print(isinstance(st.get_id(), int))
print(st.get_lineno() == 0)
print(st.is_optimized() == False)
print(st.is_nested() == False)
print(isinstance(st.has_children(), bool))
ids = list(st.get_identifiers())
print(isinstance(ids, list))
syms = st.get_symbols()
print(isinstance(syms, list))
children = st.get_children()
print(isinstance(children, list))

# ── 5. SymbolTable method presence ────────────────────────────────────────────
print(hasattr(st, 'get_type'))
print(hasattr(st, 'get_id'))
print(hasattr(st, 'get_name'))
print(hasattr(st, 'get_lineno'))
print(hasattr(st, 'is_optimized'))
print(hasattr(st, 'is_nested'))
print(hasattr(st, 'has_children'))
print(hasattr(st, 'get_identifiers'))
print(hasattr(st, 'lookup'))
print(hasattr(st, 'get_symbols'))
print(hasattr(st, 'get_children'))

# ── 6. Function class method presence ─────────────────────────────────────────
print(hasattr(symtable.Function, 'get_parameters'))
print(hasattr(symtable.Function, 'get_locals'))
print(hasattr(symtable.Function, 'get_globals'))
print(hasattr(symtable.Function, 'get_frees'))
print(hasattr(symtable.Function, 'get_nonlocals'))
print(hasattr(symtable.Function, 'get_type'))
print(hasattr(symtable.Function, 'get_symbols'))

# ── 7. Class class method presence ────────────────────────────────────────────
print(hasattr(symtable.Class, 'get_methods'))
print(hasattr(symtable.Class, 'get_type'))
print(hasattr(symtable.Class, 'get_symbols'))

# ── 8. Symbol — directly constructed ─────────────────────────────────────────
sym_ref = symtable.Symbol('x', symtable.USE)
print(isinstance(sym_ref, symtable.Symbol))
print(sym_ref.get_name() == 'x')
print(sym_ref.is_referenced() == True)
print(sym_ref.is_assigned() == False)
print(sym_ref.is_parameter() == False)
print(sym_ref.is_imported() == False)

sym_local = symtable.Symbol('y', symtable.DEF_LOCAL)
print(sym_local.is_assigned() == True)
print(sym_local.is_referenced() == False)

sym_param = symtable.Symbol('p', symtable.DEF_PARAM)
print(sym_param.is_parameter() == True)

sym_imp = symtable.Symbol('m', symtable.DEF_IMPORT)
print(sym_imp.is_imported() == True)

sym_nloc = symtable.Symbol('nl', symtable.DEF_NONLOCAL)
print(sym_nloc.is_nonlocal() == True)

# ── 9. Scope-based methods ────────────────────────────────────────────────────
sym_scoped = symtable.Symbol('z', symtable.LOCAL << symtable.SCOPE_OFF)
print(sym_scoped.is_local() == True)
print(sym_scoped.is_global() == False)
print(sym_scoped.is_free() == False)

sym_glob = symtable.Symbol('g', symtable.GLOBAL_EXPLICIT << symtable.SCOPE_OFF)
print(sym_glob.is_global() == True)
print(sym_glob.is_declared_global() == True)

sym_gimp = symtable.Symbol('gi', symtable.GLOBAL_IMPLICIT << symtable.SCOPE_OFF)
print(sym_gimp.is_global() == True)
print(sym_gimp.is_declared_global() == False)

sym_free = symtable.Symbol('fv', symtable.FREE << symtable.SCOPE_OFF)
print(sym_free.is_free() == True)
print(sym_free.is_local() == False)

# ── 10. DEF flag methods ──────────────────────────────────────────────────────
sym_fc = symtable.Symbol('fc', symtable.DEF_FREE_CLASS)
print(sym_fc.is_free_class() == True)

sym_ann = symtable.Symbol('ann', symtable.DEF_ANNOT)
print(sym_ann.is_annotated() == True)

sym_ci = symtable.Symbol('ci', symtable.DEF_COMP_ITER)
print(sym_ci.is_comp_iter() == True)

sym_cc = symtable.Symbol('cc', symtable.DEF_COMP_CELL)
print(sym_cc.is_comp_cell() == True)

sym_tp = symtable.Symbol('tp', symtable.DEF_TYPE_PARAM)
print(sym_tp.is_type_parameter() == True)

# ── 11. Namespace methods ─────────────────────────────────────────────────────
sym_ns = symtable.Symbol('ns', 0, namespaces=['tbl'])
print(sym_ns.is_namespace() == True)
print(len(sym_ns.get_namespaces()) == 1)
sym_ns2 = symtable.Symbol('ns2', 0, namespaces=['only'])
print(sym_ns2.get_namespace() == 'only')

sym_plain = symtable.Symbol('plain', 0)
print(sym_plain.is_namespace() == False)
print(len(sym_plain.get_namespaces()) == 0)

# ── 12. Symbol boolean methods on ref symbol ──────────────────────────────────
print(isinstance(sym_ref.is_annotated(), bool))
print(isinstance(sym_ref.is_comp_iter(), bool))
print(isinstance(sym_ref.is_comp_cell(), bool))
print(isinstance(sym_ref.is_type_parameter(), bool))
print(isinstance(sym_ref.is_free_class(), bool))

# ── 13. SymbolTableType ───────────────────────────────────────────────────────
print(hasattr(symtable, 'SymbolTableType'))
stt = symtable.SymbolTableType
print(stt.MODULE == 'module')
print(stt.FUNCTION == 'function')
print(stt.CLASS == 'class')
print(stt.ANNOTATION == 'annotation')

print('done')
