import ast

# ── 1. PyCF constants ─────────────────────────────────────────────────────────
print(ast.PyCF_ONLY_AST == 1024)
print(ast.PyCF_ALLOW_TOP_LEVEL_AWAIT == 8192)
print(isinstance(ast.PyCF_TYPE_COMMENTS, int))
print(isinstance(ast.PyCF_OPTIMIZED_AST, int))

# ── 2. Node classes exist ─────────────────────────────────────────────────────
print(hasattr(ast, 'AST'))
print(hasattr(ast, 'Module'))
print(hasattr(ast, 'FunctionDef'))
print(hasattr(ast, 'ClassDef'))
print(hasattr(ast, 'Constant'))
print(hasattr(ast, 'Name'))
print(hasattr(ast, 'Load'))
print(hasattr(ast, 'Store'))
print(hasattr(ast, 'Del'))
print(hasattr(ast, 'Assign'))
print(hasattr(ast, 'Return'))
print(hasattr(ast, 'If'))
print(hasattr(ast, 'For'))
print(hasattr(ast, 'While'))
print(hasattr(ast, 'Import'))
print(hasattr(ast, 'ImportFrom'))
print(hasattr(ast, 'BinOp'))
print(hasattr(ast, 'Add'))
print(hasattr(ast, 'Sub'))
print(hasattr(ast, 'Mul') or hasattr(ast, 'Mult'))

# ── 3. Node instantiation ─────────────────────────────────────────────────────
node = ast.Constant(value=42)
print(isinstance(node, ast.AST))
print(isinstance(node, ast.Constant))
print(node.value == 42)

name_node = ast.Name(id='x', ctx=ast.Load())
print(isinstance(name_node, ast.AST))
print(name_node.id == 'x')

m2 = ast.Module(body=[], type_ignores=[])
print(isinstance(m2, ast.AST))
print(isinstance(m2, ast.Module))

# ── 4. _fields attribute ──────────────────────────────────────────────────────
print(hasattr(ast.Module, '_fields'))
print(hasattr(ast.Constant, '_fields'))
print(hasattr(ast.Name, '_fields'))
print('body' in ast.Module._fields)
print('value' in ast.Constant._fields)
print('id' in ast.Name._fields)

# ── 5. literal_eval — basic types ─────────────────────────────────────────────
print(ast.literal_eval('42') == 42)
print(ast.literal_eval('3.14') == 3.14)
print(ast.literal_eval('"hello"') == 'hello')
print(ast.literal_eval("'world'") == 'world')
print(ast.literal_eval('True') == True)
print(ast.literal_eval('False') == False)
print(ast.literal_eval('None') is None)

# ── 6. literal_eval — collections ─────────────────────────────────────────────
print(ast.literal_eval('[1, 2, 3]') == [1, 2, 3])
print(ast.literal_eval('{"a": 1}') == {"a": 1})
print(ast.literal_eval('(1, 2)') == (1, 2))
print(ast.literal_eval('[]') == [])

# ── 7. parse returns Module ───────────────────────────────────────────────────
tree = ast.parse('x = 1')
print(isinstance(tree, ast.Module))
print(isinstance(tree, ast.AST))

# ── 8. dump produces a string with the class name ─────────────────────────────
dumped = ast.dump(ast.Constant(value=42))
print(isinstance(dumped, str))
print(len(dumped) > 0)
print('Constant' in dumped)

# ── 9. unparse ────────────────────────────────────────────────────────────────
unparsed = ast.unparse(ast.parse('x = 1'))
print(isinstance(unparsed, str))

# ── 10. fix_missing_locations returns the same node ──────────────────────────
n2 = ast.Constant(value=1)
result = ast.fix_missing_locations(n2)
print(result is n2)

# ── 11. copy_location ─────────────────────────────────────────────────────────
n3 = ast.Constant(value=3, lineno=5, col_offset=2, end_lineno=5, end_col_offset=3)
n4 = ast.Constant(value=4)
ast.copy_location(n4, n3)
print(n4.lineno == 5)
print(n4.col_offset == 2)

# ── 12. increment_lineno ──────────────────────────────────────────────────────
n5 = ast.Constant(value=5, lineno=1, end_lineno=1)
ast.increment_lineno(n5, 3)
print(n5.lineno == 4)

# ── 13. iter_fields ───────────────────────────────────────────────────────────
n6 = ast.Constant(value=99)
fields = list(ast.iter_fields(n6))
print(any(name == 'value' and val == 99 for name, val in fields))

# ── 14. iter_child_nodes ──────────────────────────────────────────────────────
n7 = ast.Module(body=[], type_ignores=[])
children = list(ast.iter_child_nodes(n7))
print(isinstance(children, list))

# ── 15. walk ──────────────────────────────────────────────────────────────────
n8 = ast.Constant(value=1)
walked = list(ast.walk(n8))
print(len(walked) >= 1)
print(n8 in walked)

# ── 16. NodeVisitor ───────────────────────────────────────────────────────────
print(hasattr(ast, 'NodeVisitor'))
print(hasattr(ast, 'NodeTransformer'))
visitor = ast.NodeVisitor()
print(hasattr(visitor, 'visit'))
print(hasattr(visitor, 'generic_visit'))
print(issubclass(ast.NodeTransformer, ast.NodeVisitor))

# ── 17. NodeVisitor subclassing ───────────────────────────────────────────────
class ConstVisitor(ast.NodeVisitor):
    def __init__(self):
        self.values = []
    def visit_Constant(self, node):
        self.values.append(node.value)
        self.generic_visit(node)

c_node = ast.Constant(value=77)
v2 = ConstVisitor()
v2.visit(c_node)
print(77 in v2.values)

# ── 18. Additional node classes ───────────────────────────────────────────────
print(hasattr(ast, 'AsyncFunctionDef'))
print(hasattr(ast, 'comprehension'))
print(hasattr(ast, 'ExceptHandler'))
print(hasattr(ast, 'arguments'))
print(hasattr(ast, 'arg'))
print(hasattr(ast, 'keyword'))
print(hasattr(ast, 'alias'))
print(hasattr(ast, 'Eq'))
print(hasattr(ast, 'NotEq'))
print(hasattr(ast, 'And'))
print(hasattr(ast, 'Or'))

print('done')
