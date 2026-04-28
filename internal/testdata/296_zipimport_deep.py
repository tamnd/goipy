"""Comprehensive zipimport module test — covers all public API from the Python docs."""
import zipimport
import zipfile
import tempfile
import os


# ── Module-level constants ────────────────────────────────────────────────────

print(zipimport.path_sep == "/")                             # True
print(isinstance(zipimport.alt_path_sep, str))               # True
print(zipimport.END_CENTRAL_DIR_SIZE == 22)                  # True
print(zipimport.END_CENTRAL_DIR_SIZE_64 == 56)               # True
print(zipimport.END_CENTRAL_DIR_LOCATOR_SIZE_64 == 20)       # True
print(zipimport.MAX_COMMENT_LEN == 65535)                    # True
print(zipimport.MAX_UINT32 == 4294967295)                    # True
print(zipimport.ZIP64_EXTRA_TAG == 1)                        # True
print(zipimport.STRING_END_ARCHIVE == b'PK\x05\x06')         # True
print(zipimport.STRING_END_LOCATOR_64 == b'PK\x06\x07')      # True
print(zipimport.STRING_END_ZIP_64 == b'PK\x06\x06')          # True
print(isinstance(zipimport.cp437_table, str))                # True
print(len(zipimport.cp437_table) == 256)                     # True


# ── ZipImportError ────────────────────────────────────────────────────────────

print(issubclass(zipimport.ZipImportError, ImportError))     # True
print(issubclass(zipimport.ZipImportError, Exception))       # True

try:
    raise zipimport.ZipImportError("test error")
except zipimport.ZipImportError as e:
    print(str(e) == "test error")                            # True

try:
    raise zipimport.ZipImportError("test error")
except ImportError:
    print(True)                                              # True — is-a ImportError


# ── zipimporter: invalid path raises ZipImportError ──────────────────────────

try:
    zipimport.zipimporter("/nonexistent/path.zip")
    print(False)
except zipimport.ZipImportError:
    print(True)                                              # True


# ── Create a test zip and test zipimporter ────────────────────────────────────

tmp = tempfile.mktemp(suffix=".zip")
with zipfile.ZipFile(tmp, "w") as zf:
    zf.writestr("mypkg/__init__.py", "x = 1\n")
    zf.writestr("mypkg/sub.py", "y = 2\n")
    zf.writestr("standalone.py", "z = 3\n")

try:
    zi = zipimport.zipimporter(tmp)

    # ── Attributes ──────────────────────────────────────────────────────────

    print(type(zi).__name__ == "zipimporter")                # True
    print(zi.archive == tmp)                                 # True
    print(zi.prefix == "")                                   # True
    print("zipimporter" in repr(zi))                         # True

    # ── find_spec ────────────────────────────────────────────────────────────

    spec = zi.find_spec("mypkg")
    print(spec is not None)                                  # True
    print(zi.find_spec("missing") is None)                   # True
    spec2 = zi.find_spec("standalone")
    print(spec2 is not None)                                 # True

    # ── is_package ───────────────────────────────────────────────────────────

    print(zi.is_package("mypkg") == True)                    # True
    print(zi.is_package("standalone") == False)              # True

    try:
        zi.is_package("missing")
        print(False)
    except zipimport.ZipImportError:
        print(True)                                          # True

    # ── get_filename ─────────────────────────────────────────────────────────

    fn = zi.get_filename("mypkg")
    print(isinstance(fn, str))                               # True
    print(tmp in fn)                                         # True

    try:
        zi.get_filename("missing")
        print(False)
    except zipimport.ZipImportError:
        print(True)                                          # True

    # ── get_source ───────────────────────────────────────────────────────────

    src = zi.get_source("mypkg")
    print(isinstance(src, str))                              # True
    print("x = 1" in src)                                   # True

    src2 = zi.get_source("standalone")
    print(isinstance(src2, str))                             # True
    print("z = 3" in src2)                                  # True

    try:
        zi.get_source("missing")
        print(False)
    except zipimport.ZipImportError:
        print(True)                                          # True

    # ── get_data ─────────────────────────────────────────────────────────────

    data = zi.get_data(os.path.join(tmp, "mypkg/__init__.py"))
    print(isinstance(data, bytes))                           # True
    print(b"x = 1" in data)                                 # True

    try:
        zi.get_data(os.path.join(tmp, "nonexistent.py"))
        print(False)
    except OSError:
        print(True)                                          # True

    # ── get_code ─────────────────────────────────────────────────────────────

    code_obj = zi.get_code("mypkg")
    print(code_obj is not None)                              # True

    try:
        zi.get_code("missing")
        print(False)
    except zipimport.ZipImportError:
        print(True)                                          # True

    # ── _get_files ───────────────────────────────────────────────────────────

    files = zi._get_files()
    print(isinstance(files, dict))                           # True
    print(len(files) > 0)                                    # True
    print("mypkg/__init__.py" in files)                      # True
    print("standalone.py" in files)                          # True

    # ── invalidate_caches ────────────────────────────────────────────────────

    zi.invalidate_caches()
    print(True)                                              # True

    # ── get_resource_reader ──────────────────────────────────────────────────

    rr = zi.get_resource_reader("mypkg")
    print(rr is not None)                                    # True

    # ── load_module / create_module / exec_module (stubs) ────────────────────

    print(callable(zi.load_module))                          # True
    print(callable(zi.create_module))                        # True
    print(callable(zi.exec_module))                          # True

    # ── prefix support ───────────────────────────────────────────────────────

    zi2 = zipimport.zipimporter(tmp + "/mypkg")
    print(zi2.archive == tmp)                                # True
    print(zi2.prefix == "mypkg/")                            # True
    spec3 = zi2.find_spec("sub")
    print(spec3 is not None)                                 # True

finally:
    os.unlink(tmp)

print("done")
