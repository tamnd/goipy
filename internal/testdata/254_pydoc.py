import pydoc
import io


def test_error_during_import():
    try:
        raise pydoc.ErrorDuringImport('myfile.py', (ValueError, ValueError('oops'), None))
    except pydoc.ErrorDuringImport as e:
        print(type(e).__name__)
        print('myfile.py' in str(e))
    print('test_error_during_import ok')


def test_getdoc():
    def myfunc():
        """Return the answer."""
        pass

    class MyClass:
        """A simple class."""
        pass

    print(repr(pydoc.getdoc(myfunc)))
    print(repr(pydoc.getdoc(MyClass)))

    def nodoc():
        pass
    print(repr(pydoc.getdoc(nodoc)))
    print('test_getdoc ok')


def test_describe():
    def myfunc(): pass

    class MyClass: pass

    print(pydoc.describe(myfunc))
    print(pydoc.describe(MyClass))
    print(pydoc.describe(42))
    print(pydoc.describe('hello'))
    print(pydoc.describe([1, 2]))
    print(pydoc.describe({'a': 1}))
    print(pydoc.describe((1, 2)))
    print(pydoc.describe(None))
    print('test_describe ok')


def test_splitdoc():
    print(pydoc.splitdoc('Synopsis only.'))
    print(pydoc.splitdoc('Synopsis line.\n\nDetails here.'))
    print(pydoc.splitdoc('First line.\nNo blank line.'))
    print(pydoc.splitdoc(''))
    print('test_splitdoc ok')


def test_plain():
    print(repr(pydoc.plain('a\x08b c\x08d')))
    print(repr(pydoc.plain('hello')))
    print(repr(pydoc.plain('a\x08\x08b')))
    print('test_plain ok')


def test_stripid():
    print(pydoc.stripid('<function foo at 0x7f1234abcdef>'))
    print(pydoc.stripid('no address here'))
    print(pydoc.stripid('<class bar at 0xABC>'))
    print('test_stripid ok')


def test_replace():
    print(pydoc.replace('hello world', 'world', 'earth'))
    print(pydoc.replace('aabbcc', 'aa', 'x', 'bb', 'y', 'cc', 'z'))
    print(pydoc.replace('unchanged', 'xyz', 'abc'))
    print('test_replace ok')


def test_isdata():
    def myfunc(): pass
    class MyClass: pass
    print(pydoc.isdata(42))
    print(pydoc.isdata('hello'))
    print(pydoc.isdata([1, 2]))
    print(pydoc.isdata(myfunc))
    print(pydoc.isdata(MyClass))
    print('test_isdata ok')


def test_visiblename():
    print(pydoc.visiblename('foo'))
    print(pydoc.visiblename('_private'))
    print(pydoc.visiblename('_private', all=['_private']))
    print(bool(pydoc.visiblename('__init__')))
    print(bool(pydoc.visiblename('__builtins__')))
    print(bool(pydoc.visiblename('__doc__')))
    print('test_visiblename ok')


def test_ispath():
    print(pydoc.ispath('/usr/local/lib'))
    print(pydoc.ispath('./relative'))
    print(pydoc.ispath('os.path'))
    print(pydoc.ispath('hello'))
    print('test_ispath ok')


def test_cram():
    print(pydoc.cram('hello world', 20))
    print(pydoc.cram('hello world', 8))
    print(pydoc.cram('hi', 10))
    print('test_cram ok')


def test_repr_class():
    r = pydoc.Repr()
    print(type(r).__name__)
    print(r.repr(42))
    print(r.repr('hello'))
    print(r.repr([1, 2, 3]))
    long_str = 'x' * 100
    result = r.repr(long_str)
    print(len(result) < 100)
    print('test_repr_class ok')


def test_helper():
    h = pydoc.Helper()
    print(type(h).__name__)
    print(repr(h))
    print(pydoc.help is not None)
    print('test_helper ok')


def test_render_doc():
    def myfunc():
        """Return the answer."""
        pass
    doc = pydoc.render_doc(myfunc)
    print(type(doc).__name__)
    print('myfunc' in doc)
    print('test_render_doc ok')


def test_locate():
    import os
    obj = pydoc.locate('os.path.join')
    print(obj is not None)
    obj2 = pydoc.locate('nonexistent_xyz_module_999')
    print(obj2 is None)
    print('test_locate ok')


def test_safeimport():
    import sys
    m = pydoc.safeimport('sys')
    print(m is sys)
    m2 = pydoc.safeimport('nonexistent_xyz_module_999')
    print(m2 is None)
    print('test_safeimport ok')


test_error_during_import()
test_getdoc()
test_describe()
test_splitdoc()
test_plain()
test_stripid()
test_replace()
test_isdata()
test_visiblename()
test_ispath()
test_cram()
test_repr_class()
test_helper()
test_render_doc()
test_locate()
test_safeimport()
