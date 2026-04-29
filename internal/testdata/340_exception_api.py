"""v0.0.340 exception API — add_note, with_traceback, __suppress_context__,
and BaseExceptionGroup.split / subgroup / derive."""


# ── 1. add_note populates __notes__
e = ValueError("boom")
e.add_note("first note")
e.add_note("second note")
print(e.__notes__)


# ── 2. with_traceback(None) returns self and clears the traceback
err = TypeError("t")
same = err.with_traceback(None)
print(same is err)
print(err.__traceback__ is None)


# ── 3. __suppress_context__ defaults to False, settable to True
plain = RuntimeError("r")
print(plain.__suppress_context__)
plain.__suppress_context__ = True
print(plain.__suppress_context__)


# ── 4. BaseExceptionGroup.split partitions a flat group
g = BaseExceptionGroup("g", [ValueError("v"), TypeError("t"), ValueError("v2")])
matched, rest = g.split(ValueError)
print(type(matched).__name__, len(matched.exceptions))
print(type(rest).__name__, len(rest.exceptions))
print([type(e).__name__ for e in matched.exceptions])
print([type(e).__name__ for e in rest.exceptions])


# ── 5. subgroup returns just the matching half (or None)
g2 = ExceptionGroup("g2", [ValueError("v"), TypeError("t")])
sub = g2.subgroup(TypeError)
print(type(sub).__name__, [type(e).__name__ for e in sub.exceptions])
miss = g2.subgroup(KeyError)
print(miss is None)


# ── 6. derive returns a same-class group with the supplied excs
g3 = ExceptionGroup("g3", [ValueError("v")])
derived = g3.derive([TypeError("t"), KeyError("k")])
print(type(derived).__name__, [type(e).__name__ for e in derived.exceptions])
print(derived.message)


# ── 7. split with a callable predicate
g4 = BaseExceptionGroup("g4", [ValueError(1), ValueError(99), TypeError("x")])
matched, rest = g4.split(lambda e: isinstance(e, ValueError) and e.args[0] > 10)
print([type(e).__name__ + ":" + repr(e.args[0]) for e in matched.exceptions])
print([type(e).__name__ for e in rest.exceptions])


# ── 8. add_note + raise round-trips through except*
caught = []
try:
    raise ExceptionGroup("g5", [ValueError("v"), TypeError("t")])
except* ValueError as eg:
    for inner in eg.exceptions:
        inner.add_note("seen-by-except*")
        caught.append((type(inner).__name__, inner.__notes__))
except* TypeError:
    pass
print(caught)
