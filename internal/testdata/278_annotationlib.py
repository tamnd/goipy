import annotationlib

# Format enum
print(annotationlib.Format.VALUE == 1)      # True
print(annotationlib.Format.FORWARDREF == 2) # True
print(annotationlib.Format.STRING == 3)     # True

# ForwardRef
fref = annotationlib.ForwardRef("int")
print(fref.__forward_arg__)           # int
print(fref.__forward_evaluated__)     # False
print(repr(fref))                     # ForwardRef('int')

# get_annotations on a class with annotations
class Point:
    x: int
    y: str

ann = annotationlib.get_annotations(Point)
print(isinstance(ann, dict))   # True

# get_annotations on a class without annotations
class Empty:
    pass

ann2 = annotationlib.get_annotations(Empty)
print(isinstance(ann2, dict))  # True

# annotations_from_str
d = annotationlib.annotations_from_str()
print(isinstance(d, dict))  # True

# type_repr
print(annotationlib.type_repr("int") == "int")   # True
print(annotationlib.type_repr(int) == "int")     # True

print("done")
