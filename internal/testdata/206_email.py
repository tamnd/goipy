"""Tests for the email package."""
import email
import email.message
import email.mime.text
import email.mime.multipart
import email.mime.application
import email.utils
import email.header
import email.encoders
import email.errors
import email.generator
import email.parser
import io


# ─── test_message_from_string ─────────────────────────────────────────────────

def test_message_from_string():
    raw = (
        "From: alice@example.com\n"
        "To: bob@example.com\n"
        "Subject: Hello World\n"
        "MIME-Version: 1.0\n"
        "Content-Type: text/plain; charset=utf-8\n"
        "\n"
        "Hello, Bob!\n"
    )
    msg = email.message_from_string(raw)
    assert msg["From"] == "alice@example.com", f"From={msg['From']!r}"
    assert msg["To"] == "bob@example.com"
    assert msg["Subject"] == "Hello World"
    assert isinstance(msg.get_payload(), str)
    assert "Hello" in msg.get_payload()
    print("message_from_string ok")


# ─── test_message_headers ─────────────────────────────────────────────────────

def test_message_headers():
    msg = email.message.Message()
    msg["From"] = "alice@example.com"
    msg["To"] = "bob@example.com"
    msg["CC"] = "charlie@example.com"
    msg["CC"] = "dave@example.com"  # duplicate

    # __contains__
    assert "From" in msg
    assert "from" in msg  # case-insensitive
    assert "X-Missing" not in msg

    # get returns None for missing, failobj for missing
    assert msg["X-Missing"] is None
    assert msg.get("X-Missing") is None
    assert msg.get("X-Missing", "default") == "default"

    # keys/values/items
    keys = msg.keys()
    assert "From" in keys
    assert "To" in keys

    vals = msg.values()
    assert "alice@example.com" in vals

    items = msg.items()
    assert len(items) == 4  # From, To, CC, CC

    # get_all returns all values for a header
    cc_all = msg.get_all("CC")
    assert len(cc_all) == 2
    assert "charlie@example.com" in cc_all
    assert "dave@example.com" in cc_all

    # get_all missing returns failobj
    assert msg.get_all("X-Missing") is None
    assert msg.get_all("X-Missing", []) == []

    # replace_header
    msg.replace_header("From", "new@example.com")
    assert msg["From"] == "new@example.com"

    # __delitem__
    del msg["CC"]
    assert msg.get_all("CC") is None

    # __len__
    assert len(msg) == 2  # From, To
    print("message_headers ok")


# ─── test_content_type ────────────────────────────────────────────────────────

def test_content_type():
    msg = email.message.Message()
    msg["Content-Type"] = "text/html; charset=utf-8"

    assert msg.get_content_type() == "text/html", f"ct={msg.get_content_type()!r}"
    assert msg.get_content_maintype() == "text"
    assert msg.get_content_subtype() == "html"
    assert msg.get_content_charset() == "utf-8"
    assert msg.get_default_type() == "text/plain"

    # get_param
    cs = msg.get_param("charset")
    assert cs == "utf-8", f"charset param={cs!r}"

    # get_params returns list of tuples
    params = msg.get_params()
    assert params is not None
    assert len(params) >= 1

    print("content_type ok")


# ─── test_mime_text ───────────────────────────────────────────────────────────

def test_mime_text():
    from email.mime.text import MIMEText

    msg = MIMEText("Hello, world!", "plain", "utf-8")
    assert msg.get_content_type() == "text/plain", f"ct={msg.get_content_type()!r}"
    assert msg.get_content_charset() == "utf-8"
    # payload may be plain or base64-encoded depending on charset
    payload = msg.get_payload()
    decoded = msg.get_payload(decode=True)
    if decoded is not None:
        assert b"Hello" in decoded, f"decoded payload={decoded!r}"
    else:
        assert "Hello" in payload, f"payload={payload!r}"
    assert msg["MIME-Version"] == "1.0"
    print("mime_text ok")


# ─── test_mime_multipart ──────────────────────────────────────────────────────

def test_mime_multipart():
    from email.mime.multipart import MIMEMultipart
    from email.mime.text import MIMEText

    msg = MIMEMultipart("mixed")
    assert msg.get_content_maintype() == "multipart"
    assert msg.get_content_subtype() == "mixed"
    assert msg.is_multipart()

    part1 = MIMEText("Hello, world!", "plain")
    part2 = MIMEText("<b>Hello</b>", "html")
    msg.attach(part1)
    msg.attach(part2)

    parts = msg.get_payload()
    assert isinstance(parts, list), f"payload type={type(parts)}"
    assert len(parts) == 2, f"parts count={len(parts)}"
    assert parts[0].get_content_type() == "text/plain"
    assert parts[1].get_content_type() == "text/html"
    print("mime_multipart ok")


# ─── test_utils_parseaddr ─────────────────────────────────────────────────────

def test_utils_parseaddr():
    name, addr = email.utils.parseaddr("Alice <alice@example.com>")
    assert name == "Alice", f"name={name!r}"
    assert addr == "alice@example.com", f"addr={addr!r}"

    # formataddr round-trip
    formatted = email.utils.formataddr(("Bob", "bob@example.com"))
    assert "bob@example.com" in formatted, f"formatted={formatted!r}"

    # No name
    name2, addr2 = email.utils.parseaddr("bob@example.com")
    assert addr2 == "bob@example.com", f"addr2={addr2!r}"
    print("utils_parseaddr ok")


# ─── test_utils_formatdate ────────────────────────────────────────────────────

def test_utils_formatdate():
    import time
    ts = time.time()
    datestr = email.utils.formatdate(ts)
    assert isinstance(datestr, str) and len(datestr) > 10, f"datestr={datestr!r}"
    # Should contain year
    assert "20" in datestr

    # make_msgid
    msgid = email.utils.make_msgid()
    assert "@" in msgid, f"msgid={msgid!r}"
    assert "<" in msgid and ">" in msgid
    print("utils_formatdate ok")


# ─── test_header_decode ───────────────────────────────────────────────────────

def test_header_decode():
    # Plain header — no encoding
    result = email.header.decode_header("Hello World")
    assert len(result) >= 1
    decoded, charset = result[0]
    assert charset is None
    assert "Hello" in str(decoded)

    # Base64-encoded word: =?utf-8?b?SGVsbG8gV29ybGQ=?=
    encoded = "=?utf-8?b?SGVsbG8gV29ybGQ=?="
    result2 = email.header.decode_header(encoded)
    assert len(result2) >= 1
    decoded2, charset2 = result2[0]
    assert charset2 is not None
    assert charset2.lower() == "utf-8"
    # decoded2 should be bytes containing "Hello World"
    if isinstance(decoded2, bytes):
        assert decoded2 == b"Hello World", f"decoded={decoded2!r}"
    else:
        assert "Hello" in str(decoded2)

    # make_header
    hdr = email.header.make_header(result2)
    assert hdr is not None
    print("header_decode ok")


# ─── test_errors ──────────────────────────────────────────────────────────────

def test_errors():
    # MessageError is an Exception
    try:
        raise email.errors.MessageError("test error")
    except Exception as e:
        assert "test error" in str(e)

    # HeaderParseError is a MessageParseError is a MessageError
    try:
        raise email.errors.HeaderParseError("bad header")
    except email.errors.MessageError:
        pass  # should be caught

    # MultipartConversionError is both MessageError and TypeError
    try:
        raise email.errors.MultipartConversionError("bad conversion")
    except TypeError:
        pass  # should be caught as TypeError

    print("errors ok")


# ─── test_generator ───────────────────────────────────────────────────────────

def test_generator():
    from email.mime.text import MIMEText
    from email.generator import Generator

    msg = MIMEText("Hello from generator!", "plain")
    msg["From"] = "alice@example.com"
    msg["To"] = "bob@example.com"

    # as_string() works
    s = msg.as_string()
    assert isinstance(s, str)
    assert "From" in s or "alice" in s
    assert "Content-Type" in s

    # Generator.flatten writes to StringIO
    fp = io.StringIO()
    gen = Generator(fp)
    gen.flatten(msg)
    out = fp.getvalue()
    assert isinstance(out, str) and len(out) > 0
    assert "Content-Type" in out
    print("generator ok")


# ─── test_walk ────────────────────────────────────────────────────────────────

def test_walk():
    from email.mime.multipart import MIMEMultipart
    from email.mime.text import MIMEText

    msg = MIMEMultipart("mixed")
    msg.attach(MIMEText("plain text", "plain"))
    msg.attach(MIMEText("<b>html</b>", "html"))

    parts = list(msg.walk())
    assert len(parts) == 3, f"walk() yielded {len(parts)} parts, expected 3"
    # First is the multipart itself
    assert parts[0].get_content_maintype() == "multipart"
    assert parts[1].get_content_type() == "text/plain"
    assert parts[2].get_content_type() == "text/html"
    print("walk ok")


# ─── test_encoders ────────────────────────────────────────────────────────────

def test_encoders():
    from email.mime.application import MIMEApplication
    from email.encoders import encode_noop, encode_base64
    import email.message

    # encode_noop does nothing
    msg = email.message.Message()
    msg.set_payload(b"raw bytes")
    encode_noop(msg)  # should not raise

    # encode_base64 on a MIMEApplication
    app = MIMEApplication(b"binary data", _subtype="octet-stream", _encoder=encode_noop)
    encode_base64(app)
    cte = app["Content-Transfer-Encoding"]
    assert cte == "base64", f"CTE after encode_base64={cte!r}"
    # payload should be base64-encoded
    payload = app.get_payload()
    assert isinstance(payload, str)

    print("encoders ok")


if __name__ == "__main__":
    test_message_from_string()
    test_message_headers()
    test_content_type()
    test_mime_text()
    test_mime_multipart()
    test_utils_parseaddr()
    test_utils_formatdate()
    test_header_decode()
    test_errors()
    test_generator()
    test_walk()
    test_encoders()
    print("ALL OK")
