import json
import io

def test_dumps_basic():
    assert json.dumps(None) == "null"
    assert json.dumps(True) == "true"
    assert json.dumps(False) == "false"
    assert json.dumps(42) == "42"
    assert json.dumps(3.14) == "3.14"
    assert json.dumps("hello") == '"hello"'
    assert json.dumps([1, 2, 3]) == "[1, 2, 3]"
    assert json.dumps({"a": 1}) == '{"a": 1}'
    assert json.dumps((1, 2)) == "[1, 2]"
    print("test_dumps_basic ok")

def test_dumps_indent():
    s = json.dumps({"b": 2, "a": 1}, indent=2, sort_keys=True)
    expected = '{\n  "a": 1,\n  "b": 2\n}'
    assert s == expected, repr(s)
    print("test_dumps_indent ok")

def test_dumps_separators():
    s = json.dumps([1, 2, 3], separators=(",", ":"))
    assert s == "[1,2,3]", repr(s)
    s2 = json.dumps({"a": 1}, separators=(",", ":"))
    assert s2 == '{"a":1}', repr(s2)
    print("test_dumps_separators ok")

def test_dumps_ensure_ascii():
    s = json.dumps("café", ensure_ascii=False)
    assert s == '"café"', repr(s)
    s2 = json.dumps("café", ensure_ascii=True)
    assert s2 == '"caf\\u00e9"', repr(s2)
    print("test_dumps_ensure_ascii ok")

def test_dumps_allow_nan():
    import math
    s = json.dumps(math.nan, allow_nan=True)
    assert s == "NaN", repr(s)
    s2 = json.dumps(math.inf, allow_nan=True)
    assert s2 == "Infinity", repr(s2)
    try:
        json.dumps(math.nan, allow_nan=False)
        assert False, "should raise"
    except (ValueError, TypeError):
        pass
    print("test_dumps_allow_nan ok")

def test_dumps_skipkeys():
    # tuple keys are non-basic and get skipped when skipkeys=True
    d = {"str": "str_key", (1, 2): "tuple_key"}
    s = json.dumps(d, skipkeys=True)
    parsed = json.loads(s)
    assert "str" in parsed
    assert len(parsed) == 1
    print("test_dumps_skipkeys ok")

def test_dumps_default():
    class MyObj:
        def __init__(self, v):
            self.v = v
    def my_default(o):
        if isinstance(o, MyObj):
            return {"myobj": o.v}
        raise TypeError(f"not serializable: {type(o)}")
    s = json.dumps(MyObj(42), default=my_default)
    assert s == '{"myobj": 42}', repr(s)
    print("test_dumps_default ok")

def test_loads_basic():
    assert json.loads("null") is None
    assert json.loads("true") is True
    assert json.loads("false") is False
    assert json.loads("42") == 42
    assert json.loads("3.14") == 3.14
    assert json.loads('"hello"') == "hello"
    assert json.loads("[1, 2]") == [1, 2]
    assert json.loads('{"a": 1}') == {"a": 1}
    # bytes and bytearray
    assert json.loads(b'"bytes"') == "bytes"
    assert json.loads(bytearray(b'"ba"')) == "ba"
    print("test_loads_basic ok")

def test_loads_object_hook():
    def hook(d):
        return {k.upper(): v for k, v in d.items()}
    result = json.loads('{"x": 1, "y": 2}', object_hook=hook)
    assert result == {"X": 1, "Y": 2}, result
    print("test_loads_object_hook ok")

def test_loads_parse_types():
    result = json.loads("3.14", parse_float=str)
    assert result == "3.14", result
    result2 = json.loads("42", parse_int=str)
    assert result2 == "42", result2
    print("test_loads_parse_types ok")

def test_json_decode_error():
    try:
        json.loads("{invalid}")
        assert False, "should raise"
    except json.JSONDecodeError as e:
        assert e.doc == "{invalid}"
        assert e.pos >= 0
        assert e.lineno >= 1
        assert e.colno >= 1
        assert isinstance(e, ValueError)
    print("test_json_decode_error ok")

def test_json_encoder_decoder():
    enc = json.JSONEncoder(sort_keys=True, indent=None, separators=(", ", ": "))
    s = enc.encode({"b": 2, "a": 1})
    parsed = json.loads(s)
    assert parsed["a"] == 1 and parsed["b"] == 2

    dec = json.JSONDecoder()
    obj = dec.decode('{"x": 99}')
    assert obj["x"] == 99

    # raw_decode
    obj2, idx = dec.raw_decode('{"y": 7}  extra')
    assert obj2["y"] == 7
    assert idx >= 8
    print("test_json_encoder_decoder ok")

def test_dump_load():
    buf = io.StringIO()
    json.dump({"key": [1, 2, 3]}, buf)
    buf.seek(0)
    result = json.load(buf)
    assert result == {"key": [1, 2, 3]}
    # load from BytesIO (contains JSON bytes)
    bbuf = io.BytesIO(b'[1, 2]')
    result2 = json.load(bbuf)
    assert result2 == [1, 2]
    print("test_dump_load ok")

test_dumps_basic()
test_dumps_indent()
test_dumps_separators()
test_dumps_ensure_ascii()
test_dumps_allow_nan()
test_dumps_skipkeys()
test_dumps_default()
test_loads_basic()
test_loads_object_hook()
test_loads_parse_types()
test_json_decode_error()
test_json_encoder_decoder()
test_dump_load()
print("ALL OK")
