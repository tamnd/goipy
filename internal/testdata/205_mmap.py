"""Tests for mmap module."""
import mmap
import os

_TMPFILE = "/tmp/goipy_mmap_test.bin"


def _make_tmpfile(content):
    fd = os.open(_TMPFILE, os.O_CREAT | os.O_RDWR | os.O_TRUNC, 0o600)
    os.write(fd, content)
    os.close(fd)
    return os.open(_TMPFILE, os.O_RDWR)


def test_constants():
    assert mmap.ACCESS_DEFAULT == 0
    assert mmap.ACCESS_READ == 1
    assert mmap.ACCESS_WRITE == 2
    assert mmap.ACCESS_COPY == 3
    assert mmap.PROT_READ == 1
    assert mmap.PROT_WRITE == 2
    assert isinstance(mmap.PAGESIZE, int) and mmap.PAGESIZE > 0
    assert isinstance(mmap.ALLOCATIONGRANULARITY, int) and mmap.ALLOCATIONGRANULARITY > 0
    assert mmap.MAP_SHARED == 1
    assert mmap.MAP_PRIVATE == 2
    print("constants ok")


def test_anonymous():
    m = mmap.mmap(-1, 4096)
    m.write(b"hello world")
    m.seek(0)
    data = m.read(11)
    assert data == b"hello world", f"got {data!r}"
    m.close()
    assert m.closed
    print("anonymous ok")


def test_file_basic():
    content = b"Hello, mmap world!!"
    fd = _make_tmpfile(content)
    m = mmap.mmap(fd, len(content))
    os.close(fd)
    data = m.read(5)
    assert data == b"Hello", f"read={data!r}"
    assert m.tell() == 5, f"tell={m.tell()}"
    m.seek(7)
    assert m.tell() == 7
    m.write(b"mmap ")
    assert m.size() == len(content), f"size={m.size()}"
    m.close()
    os.unlink(_TMPFILE)
    print("file_basic ok")


def test_seek_modes():
    m = mmap.mmap(-1, 100)
    m.seek(10)
    assert m.tell() == 10
    m.seek(5, 1)  # SEEK_CUR
    assert m.tell() == 15, f"tell={m.tell()}"
    m.seek(-10, 2)  # SEEK_END
    assert m.tell() == 90, f"tell={m.tell()}"
    m.seek(0, 0)  # SEEK_SET
    assert m.tell() == 0
    m.close()
    print("seek_modes ok")


def test_slicing():
    m = mmap.mmap(-1, 16)
    m.write(b"abcdefghijklmnop")
    assert m[0] == ord('a'), f"m[0]={m[0]}"
    assert m[3] == ord('d')
    assert m[0:4] == b"abcd", f"m[0:4]={m[0:4]!r}"
    assert m[4:8] == b"efgh"
    m[0:4] = b"ABCD"
    assert m[0:4] == b"ABCD", f"after assign: {m[0:4]!r}"
    m.close()
    print("slicing ok")


def test_find_rfind():
    m = mmap.mmap(-1, 32)
    m.write(b"hello world hello")
    assert m.find(b"hello", 0) == 0, f"find={m.find(b'hello', 0)}"
    assert m.find(b"world", 0) == 6, f"find={m.find(b'world', 0)}"
    assert m.find(b"xyz", 0) == -1
    assert m.rfind(b"hello", 0) == 12, f"rfind={m.rfind(b'hello', 0)}"
    assert m.rfind(b"xyz", 0) == -1
    m.close()
    print("find_rfind ok")


def test_readline():
    m = mmap.mmap(-1, 32)
    m.write(b"line1\nline2\nline3")
    m.seek(0)
    line = m.readline()
    assert line == b"line1\n", f"readline={line!r}"
    line2 = m.readline()
    assert line2 == b"line2\n", f"readline2={line2!r}"
    m.close()
    print("readline ok")


def test_flush():
    content = b"flush test data!"
    fd = _make_tmpfile(content)
    m = mmap.mmap(fd, len(content))
    os.close(fd)
    m.write(b"FLUSH")
    m.flush()
    m.close()
    os.unlink(_TMPFILE)
    print("flush ok")


def test_madvise():
    m = mmap.mmap(-1, 4096)
    m.madvise(mmap.MADV_SEQUENTIAL)
    m.close()
    print("madvise ok")


def test_access_read():
    content = b"readonly data!!!"
    fd = _make_tmpfile(content)
    m = mmap.mmap(fd, len(content), access=mmap.ACCESS_READ)
    os.close(fd)
    try:
        m.write(b"fail")
        assert False, "write should have raised TypeError"
    except TypeError:
        pass
    try:
        m.write_byte(65)
        assert False, "write_byte should have raised TypeError"
    except TypeError:
        pass
    m.close()
    os.unlink(_TMPFILE)
    print("access_read ok")


def test_context_manager():
    m = mmap.mmap(-1, 1024)
    with m as mm:
        mm.write(b"context")
        mm.seek(0)
        assert mm.read(7) == b"context"
    assert m.closed
    print("context_manager ok")


if __name__ == "__main__":
    test_constants()
    test_anonymous()
    test_file_basic()
    test_seek_modes()
    test_slicing()
    test_find_rfind()
    test_readline()
    test_flush()
    test_madvise()
    test_access_read()
    test_context_manager()
    print("ALL OK")
