import mailbox, tempfile, os

def test_mbox_basic():
    with tempfile.TemporaryDirectory() as d:
        path = os.path.join(d, "test.mbox")
        mb = mailbox.mbox(path)
        key = mb.add("From: alice@example.com\n\nHello\n")
        assert len(mb) == 1
        assert key == 0
        msg = mb[key]
        assert msg["From"] == "alice@example.com"
        mb.close()
    print("mbox_basic ok")

def test_mbox_multi():
    with tempfile.TemporaryDirectory() as d:
        path = os.path.join(d, "test.mbox")
        mb = mailbox.mbox(path)
        k1 = mb.add("From: a@x.com\n\nMsg1\n")
        k2 = mb.add("From: b@x.com\n\nMsg2\n")
        assert len(mb) == 2
        assert list(mb.keys()) == [0, 1]
        mb.remove(k1)
        assert len(mb) == 1
        mb.flush()
        mb.close()
    print("mbox_multi ok")

def test_mbox_discard():
    with tempfile.TemporaryDirectory() as d:
        path = os.path.join(d, "test.mbox")
        mb = mailbox.mbox(path)
        mb.add("From: x@y.com\n\ntest\n")
        mb.discard(99)  # no error
        mb.close()
    print("mbox_discard ok")

def test_mbox_contains():
    with tempfile.TemporaryDirectory() as d:
        path = os.path.join(d, "test.mbox")
        mb = mailbox.mbox(path)
        k = mb.add("From: z@w.com\n\nbody\n")
        assert k in mb
        assert 99 not in mb
        mb.close()
    print("mbox_contains ok")

def test_mbox_clear():
    with tempfile.TemporaryDirectory() as d:
        path = os.path.join(d, "test.mbox")
        mb = mailbox.mbox(path)
        mb.add("From: a@b.com\n\nhi\n")
        mb.add("From: c@d.com\n\nbye\n")
        mb.clear()
        assert len(mb) == 0
        mb.close()
    print("mbox_clear ok")

def test_maildir_basic():
    with tempfile.TemporaryDirectory() as d:
        mb = mailbox.Maildir(d)
        key = mb.add("From: alice@example.com\nSubject: test\n\nHello\n")
        assert len(mb) == 1
        assert key in mb
        msg = mb[key]
        assert msg["From"] == "alice@example.com"
        mb.close()
    print("maildir_basic ok")

def test_maildir_remove():
    with tempfile.TemporaryDirectory() as d:
        mb = mailbox.Maildir(d)
        k1 = mb.add("From: a@b.com\n\nfirst\n")
        k2 = mb.add("From: c@d.com\n\nsecond\n")
        assert len(mb) == 2
        mb.remove(k1)
        assert len(mb) == 1
        assert k2 in mb
        mb.close()
    print("maildir_remove ok")

def test_maildir_discard():
    with tempfile.TemporaryDirectory() as d:
        mb = mailbox.Maildir(d)
        mb.add("From: x@y.com\n\nbody\n")
        mb.discard("nonexistent_key")
        mb.close()
    print("maildir_discard ok")

def test_errors():
    try:
        raise mailbox.Error("base error")
    except mailbox.Error as e:
        assert str(e) == "base error"
    try:
        raise mailbox.NoSuchMailboxError("no mailbox")
    except mailbox.Error:
        pass
    try:
        raise mailbox.FormatError("bad format")
    except mailbox.Error:
        pass
    print("errors ok")

def test_mbox_get_string_bytes():
    with tempfile.TemporaryDirectory() as d:
        path = os.path.join(d, "test.mbox")
        mb = mailbox.mbox(path)
        k = mb.add("From: test@test.com\nSubject: hi\n\nbody text\n")
        s = mb.get_string(k)
        assert "From: test@test.com" in s
        b = mb.get_bytes(k)
        assert isinstance(b, bytes)
        assert b"From: test@test.com" in b
        mb.close()
    print("mbox_get_string_bytes ok")

test_mbox_basic()
test_mbox_multi()
test_mbox_discard()
test_mbox_contains()
test_mbox_clear()
test_maildir_basic()
test_maildir_remove()
test_maildir_discard()
test_errors()
test_mbox_get_string_bytes()
print("ALL OK")
