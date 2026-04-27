import sys
import faulthandler


def test_flags_attrs():
    f = sys.flags
    print(type(f).__name__)
    print(f.debug)
    print(f.inspect)
    print(f.interactive)
    print(f.optimize)
    print(f.dont_write_bytecode)
    print(f.no_user_site)
    print(f.no_site)
    print(f.ignore_environment)
    print(f.verbose)
    print(f.bytes_warning)
    print(f.quiet)
    print(f.isolated)
    print(f.dev_mode)
    print(f.utf8_mode)
    print(f.warn_default_encoding)
    print(f.safe_path)
    print(f.int_max_str_digits)
    print('test_flags_attrs ok')


def test_flags_sequence():
    f = sys.flags
    print(len(f))
    print(f[0])    # debug
    print(f[13])   # dev_mode
    print(f[-1])   # int_max_str_digits
    vals = tuple(f)
    print(type(vals).__name__)
    print(len(vals))
    print(vals[0])
    print(vals[13])
    print('test_flags_sequence ok')


def test_flags_devmode_guard():
    if not sys.flags.dev_mode:
        print('not in dev mode')
    if sys.flags.verbose == 0:
        print('not verbose')
    if sys.flags.optimize == 0:
        print('not optimized')
    print('test_flags_devmode_guard ok')


def test_warnoptions():
    print(type(sys.warnoptions).__name__)
    print(len(sys.warnoptions))
    print('test_warnoptions ok')


def test_xoptions():
    print(type(sys._xoptions).__name__)
    print(len(sys._xoptions))
    print('dev' in sys._xoptions)
    print('test_xoptions ok')


def test_getdefaultencoding():
    enc = sys.getdefaultencoding()
    print(type(enc).__name__)
    print(enc)
    print('test_getdefaultencoding ok')


def test_getfilesystemencoding():
    enc = sys.getfilesystemencoding()
    print(type(enc).__name__)
    print(enc)
    errs = sys.getfilesystemencodeerrors()
    print(type(errs).__name__)
    print('test_getfilesystemencoding ok')


def test_intern():
    s = sys.intern('hello')
    print(type(s).__name__)
    print(s)
    s2 = sys.intern('world')
    print(s2)
    print('test_intern ok')


def test_audit():
    sys.addaudithook(lambda event, args: None)
    sys.audit('test.event', 1, 2, 3)
    sys.audit('another.event')
    print('test_audit ok')


def test_getsizeof():
    n = sys.getsizeof(42)
    print(type(n).__name__)
    print(n > 0)
    n2 = sys.getsizeof('hello')
    print(n2 > 0)
    print('test_getsizeof ok')


def test_maxunicode():
    print(sys.maxunicode)
    print('test_maxunicode ok')


def test_is_finalizing():
    print(sys.is_finalizing())
    print('test_is_finalizing ok')


def test_faulthandler():
    print(type(faulthandler).__name__)
    print(faulthandler.is_enabled())
    faulthandler.enable()
    faulthandler.disable()
    print('test_faulthandler ok')


test_flags_attrs()
test_flags_sequence()
test_flags_devmode_guard()
test_warnoptions()
test_xoptions()
test_getdefaultencoding()
test_getfilesystemencoding()
test_intern()
test_audit()
test_getsizeof()
test_maxunicode()
test_is_finalizing()
test_faulthandler()
