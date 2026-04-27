import zipapp
import tempfile
import os
import zipfile


def run(name, fn):
    try:
        fn()
        print(f"{name}: OK")
    except Exception as e:
        print(f"{name}: FAIL ({e})")


# ── import ────────────────────────────────────────────────────────────────────

def test_import():
    import zipapp as z
    assert z is not None

run("test_import", test_import)


# ── create_archive from directory ─────────────────────────────────────────────

def test_create_archive_basic():
    with tempfile.TemporaryDirectory() as d:
        # create a directory with __main__.py
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("print('hello')\n")
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target)
        assert os.path.exists(target)

run("test_create_archive_basic", test_create_archive_basic)


def test_create_archive_is_valid_zip():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("pass\n")
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target)
        assert zipfile.is_zipfile(target)

run("test_create_archive_is_valid_zip", test_create_archive_is_valid_zip)


def test_create_archive_contains_main():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("pass\n")
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target)
        with zipfile.ZipFile(target) as zf:
            names = zf.namelist()
        assert "__main__.py" in names

run("test_create_archive_contains_main", test_create_archive_contains_main)


def test_create_archive_multiple_files():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        for name in ("__main__.py", "helper.py", "utils.py"):
            with open(os.path.join(src, name), "w") as f:
                f.write("pass\n")
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target)
        with zipfile.ZipFile(target) as zf:
            names = zf.namelist()
        assert "__main__.py" in names
        assert "helper.py" in names

run("test_create_archive_multiple_files", test_create_archive_multiple_files)


# ── interpreter / shebang ─────────────────────────────────────────────────────

def test_create_archive_with_interpreter():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("pass\n")
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target, interpreter="/usr/bin/env python3")
        assert os.path.exists(target)

run("test_create_archive_with_interpreter", test_create_archive_with_interpreter)


def test_create_archive_shebang_content():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("pass\n")
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target, interpreter="/usr/bin/env python3")
        with open(target, "rb") as f:
            first_line = f.readline()
        assert first_line.startswith(b"#!")
        assert b"python" in first_line

run("test_create_archive_shebang_content", test_create_archive_shebang_content)


def test_create_archive_no_shebang_still_valid_zip():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("pass\n")
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target)  # no interpreter
        assert zipfile.is_zipfile(target)

run("test_create_archive_no_shebang_still_valid_zip", test_create_archive_no_shebang_still_valid_zip)


# ── get_interpreter ───────────────────────────────────────────────────────────

def test_get_interpreter_none_when_no_shebang():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("pass\n")
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target)
        result = zipapp.get_interpreter(target)
        assert result is None, f"expected None, got {result!r}"

run("test_get_interpreter_none_when_no_shebang", test_get_interpreter_none_when_no_shebang)


def test_get_interpreter_returns_string():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("pass\n")
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target, interpreter="/usr/bin/env python3")
        result = zipapp.get_interpreter(target)
        assert isinstance(result, str), f"expected str, got {type(result)}"

run("test_get_interpreter_returns_string", test_get_interpreter_returns_string)


def test_get_interpreter_roundtrip():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("pass\n")
        target = os.path.join(d, "myapp.pyz")
        interp = "/usr/bin/env python3"
        zipapp.create_archive(src, target, interpreter=interp)
        result = zipapp.get_interpreter(target)
        assert result == interp, f"expected {interp!r}, got {result!r}"

run("test_get_interpreter_roundtrip", test_get_interpreter_roundtrip)


# ── auto-target ───────────────────────────────────────────────────────────────

def test_create_archive_auto_target():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("pass\n")
        zipapp.create_archive(src)  # target = src + ".pyz"
        assert os.path.exists(src + ".pyz")

run("test_create_archive_auto_target", test_create_archive_auto_target)


# ── main= param ───────────────────────────────────────────────────────────────

def test_create_archive_with_main():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        # no __main__.py; provide via main=
        with open(os.path.join(src, "mymod.py"), "w") as f:
            f.write("def run(): pass\n")
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target, main="mymod:run")
        with zipfile.ZipFile(target) as zf:
            names = zf.namelist()
        assert "__main__.py" in names

run("test_create_archive_with_main", test_create_archive_with_main)


def test_create_archive_main_content():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "mymod.py"), "w") as f:
            f.write("def run(): pass\n")
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target, main="mymod:run")
        with zipfile.ZipFile(target) as zf:
            content = zf.read("__main__.py").decode()
        assert "mymod" in content

run("test_create_archive_main_content", test_create_archive_main_content)


# ── compressed ────────────────────────────────────────────────────────────────

def test_create_archive_compressed():
    with tempfile.TemporaryDirectory() as d:
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("pass\n" * 100)
        target = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, target, compressed=True)
        assert zipfile.is_zipfile(target)

run("test_create_archive_compressed", test_create_archive_compressed)


# ── copy existing archive ─────────────────────────────────────────────────────

def test_create_archive_copy_existing():
    with tempfile.TemporaryDirectory() as d:
        # first create an archive
        src = os.path.join(d, "myapp")
        os.makedirs(src)
        with open(os.path.join(src, "__main__.py"), "w") as f:
            f.write("pass\n")
        archive1 = os.path.join(d, "myapp.pyz")
        zipapp.create_archive(src, archive1)
        # now copy it with a new interpreter
        archive2 = os.path.join(d, "myapp2.pyz")
        zipapp.create_archive(archive1, archive2, interpreter="/usr/bin/python3")
        assert zipfile.is_zipfile(archive2)
        result = zipapp.get_interpreter(archive2)
        assert result == "/usr/bin/python3"

run("test_create_archive_copy_existing", test_create_archive_copy_existing)
