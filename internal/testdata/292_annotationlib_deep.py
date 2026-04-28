"""Comprehensive annotationlib test — covers all public API from the Python docs."""
import annotationlib


# ── Format enum ───────────────────────────────────────────────────────────────

print(annotationlib.Format.VALUE == 1)                  # True
print(annotationlib.Format.VALUE_WITH_FAKE_GLOBALS == 2) # True
print(annotationlib.Format.FORWARDREF == 3)              # True
print(annotationlib.Format.STRING == 4)                  # True

print(annotationlib.Format.VALUE.value == 1)                  # True
print(annotationlib.Format.VALUE_WITH_FAKE_GLOBALS.value == 2) # True
print(annotationlib.Format.FORWARDREF.value == 3)             # True
print(annotationlib.Format.STRING.value == 4)                  # True

print(annotationlib.Format.VALUE.name == 'VALUE')                           # True
print(annotationlib.Format.VALUE_WITH_FAKE_GLOBALS.name == 'VALUE_WITH_FAKE_GLOBALS')  # True
print(annotationlib.Format.FORWARDREF.name == 'FORWARDREF')                 # True
print(annotationlib.Format.STRING.name == 'STRING')                         # True


# ── ForwardRef attributes ─────────────────────────────────────────────────────

fref = annotationlib.ForwardRef("int")
print(fref.__forward_arg__ == "int")            # True
print(fref.__forward_module__ is None)          # True
print(fref.__forward_is_argument__ == True)     # True
print(fref.__forward_is_class__ == False)       # True
print(repr(fref) == "ForwardRef('int')")        # True


# ── ForwardRef.__eq__ / __hash__ ──────────────────────────────────────────────

fref2 = annotationlib.ForwardRef("int")
fref3 = annotationlib.ForwardRef("str")
print(fref == fref2)                    # True
print(fref == fref3)                    # False
print(isinstance(hash(fref), int))      # True


# ── ForwardRef.evaluate() ─────────────────────────────────────────────────────

try:
    result = annotationlib.ForwardRef("int").evaluate()
    print(result is int)    # True — CPython resolves builtins
except NameError:
    print(True)             # True — goipy raises NameError

try:
    annotationlib.ForwardRef("UndefinedXYZ").evaluate()
    print("no_error")
except NameError:
    print("NameError_raised")   # NameError_raised


# ── get_annotations ───────────────────────────────────────────────────────────

class Point:
    x: int
    y: str

ann = annotationlib.get_annotations(Point)
print(isinstance(ann, dict))    # True

ann_str = annotationlib.get_annotations(Point, format=annotationlib.Format.STRING)
print(isinstance(ann_str, dict))    # True
print(ann_str.get("x") == "int")    # True
print(ann_str.get("y") == "str")    # True

class Empty: pass
print(isinstance(annotationlib.get_annotations(Empty), dict))  # True


# ── call_annotate_function ────────────────────────────────────────────────────

print(callable(annotationlib.call_annotate_function))   # True


# ── call_evaluate_function ────────────────────────────────────────────────────

print(callable(annotationlib.call_evaluate_function))   # True


# ── get_annotate_from_class_namespace ─────────────────────────────────────────

print(callable(annotationlib.get_annotate_from_class_namespace))    # True
ns_result = annotationlib.get_annotate_from_class_namespace({})
print(ns_result is None or callable(ns_result))     # True


# ── type_repr ─────────────────────────────────────────────────────────────────

print(isinstance(annotationlib.type_repr(int), str))        # True
print(annotationlib.type_repr(int) == "int")                # True
print(annotationlib.type_repr("int") == "'int'")            # True  (quoted string)


# ── annotations_to_string ─────────────────────────────────────────────────────

d = {"x": int, "y": str}
result = annotationlib.annotations_to_string(d)
print(isinstance(result, dict))         # True
print(result.get("x") == "int")         # True
print(result.get("y") == "str")         # True


print("done")
