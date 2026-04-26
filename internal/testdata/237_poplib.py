import poplib


def test_constants():
    assert poplib.POP3_PORT == 110
    assert poplib.POP3_SSL_PORT == 995
    print("constants ok")


def test_exception_is_exception():
    assert issubclass(poplib.error_proto, Exception)
    print("exception_is_exception ok")


def test_exception_raise_catch():
    try:
        raise poplib.error_proto("-ERR bad command")
    except poplib.error_proto as e:
        assert "-ERR" in str(e)
    try:
        raise poplib.error_proto("-ERR bad command")
    except Exception as e:
        assert "-ERR" in str(e)
    print("exception_raise_catch ok")


def test_exception_distinct():
    try:
        raise poplib.error_proto("oops")
    except ValueError:
        assert False, "error_proto should not be caught as ValueError"
    except poplib.error_proto:
        pass
    print("exception_distinct ok")


def test_pop3ssl_subclass():
    assert issubclass(poplib.POP3_SSL, poplib.POP3)
    print("pop3ssl_subclass ok")


def test_module_exports():
    assert poplib.POP3 is not None
    assert poplib.POP3_SSL is not None
    assert poplib.error_proto is not None
    assert callable(poplib.POP3)
    assert callable(poplib.POP3_SSL)
    print("module_exports ok")


test_constants()
test_exception_is_exception()
test_exception_raise_catch()
test_exception_distinct()
test_pop3ssl_subclass()
test_module_exports()
