import imaplib


def test_constants():
    assert imaplib.IMAP4_PORT == 143
    assert imaplib.IMAP4_SSL_PORT == 993
    assert imaplib.Debug == 0
    print("constants ok")


def test_exception_hierarchy():
    assert issubclass(imaplib.IMAP4.error, Exception)
    assert issubclass(imaplib.IMAP4.abort, imaplib.IMAP4.error)
    assert issubclass(imaplib.IMAP4.readonly, imaplib.IMAP4.abort)
    assert issubclass(imaplib.IMAP4.readonly, imaplib.IMAP4.error)
    assert issubclass(imaplib.IMAP4.readonly, Exception)
    print("exception_hierarchy ok")


def test_exception_raise_catch():
    try:
        raise imaplib.IMAP4.error("connection failed")
    except imaplib.IMAP4.error as e:
        assert "connection failed" in str(e)
    try:
        raise imaplib.IMAP4.abort("server error")
    except imaplib.IMAP4.error as e:
        assert "server error" in str(e)
    try:
        raise imaplib.IMAP4.readonly("mailbox readonly")
    except imaplib.IMAP4.error as e:
        assert "mailbox readonly" in str(e)
    try:
        raise imaplib.IMAP4.error("base error")
    except Exception as e:
        assert "base error" in str(e)
    print("exception_raise_catch ok")


def test_exception_distinct():
    try:
        raise imaplib.IMAP4.error("oops")
    except ValueError:
        assert False, "IMAP4.error should not be caught as ValueError"
    except imaplib.IMAP4.error:
        pass
    print("exception_distinct ok")


def test_parse_flags():
    r = imaplib.ParseFlags(b'FLAGS (\\Seen \\Recent)')
    assert r == (b'\\Seen', b'\\Recent'), repr(r)
    r2 = imaplib.ParseFlags(b'nothing here')
    assert r2 == (), repr(r2)
    r3 = imaplib.ParseFlags(b'FLAGS ()')
    assert r3 == (), repr(r3)
    print("parse_flags ok")


def test_int2ap():
    assert imaplib.Int2AP(0) == b'', repr(imaplib.Int2AP(0))
    assert imaplib.Int2AP(1) == b'B', repr(imaplib.Int2AP(1))
    assert imaplib.Int2AP(15) == b'P', repr(imaplib.Int2AP(15))
    assert imaplib.Int2AP(16) == b'BA', repr(imaplib.Int2AP(16))
    assert imaplib.Int2AP(256) == b'BAA', repr(imaplib.Int2AP(256))
    print("int2ap ok")


def test_time2internaldate():
    r = imaplib.Time2Internaldate(0)
    assert isinstance(r, str), type(r)
    assert r.startswith('"'), repr(r)
    assert r.endswith('"'), repr(r)
    print("time2internaldate ok")


def test_internaldate2tuple():
    r = imaplib.Internaldate2tuple(b'INTERNALDATE " 1-Jan-1970 00:00:00 +0000"')
    assert r is not None, repr(r)
    r2 = imaplib.Internaldate2tuple(b'bad data')
    assert r2 is None, repr(r2)
    print("internaldate2tuple ok")


def test_class_hierarchy():
    assert issubclass(imaplib.IMAP4_SSL, imaplib.IMAP4)
    assert issubclass(imaplib.IMAP4_stream, imaplib.IMAP4)
    print("class_hierarchy ok")


def test_module_exports():
    assert imaplib.IMAP4 is not None
    assert imaplib.IMAP4_SSL is not None
    assert imaplib.IMAP4_stream is not None
    assert callable(imaplib.IMAP4)
    assert callable(imaplib.ParseFlags)
    assert callable(imaplib.Int2AP)
    assert callable(imaplib.Time2Internaldate)
    assert callable(imaplib.Internaldate2tuple)
    print("module_exports ok")


test_constants()
test_exception_hierarchy()
test_exception_raise_catch()
test_exception_distinct()
test_parse_flags()
test_int2ap()
test_time2internaldate()
test_internaldate2tuple()
test_class_hierarchy()
test_module_exports()
