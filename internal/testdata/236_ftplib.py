import ftplib
import io


def test_constants():
    assert ftplib.FTP_PORT == 21
    assert ftplib.MAXLINE == 8192
    assert ftplib.MSG_OOB == 1
    assert ftplib.CRLF == "\r\n"
    assert ftplib.B_CRLF == b"\r\n"
    print("constants ok")


def test_exception_hierarchy():
    assert issubclass(ftplib.Error, Exception)
    assert issubclass(ftplib.error_reply, ftplib.Error)
    assert issubclass(ftplib.error_temp, ftplib.Error)
    assert issubclass(ftplib.error_perm, ftplib.Error)
    assert issubclass(ftplib.error_proto, ftplib.Error)
    assert issubclass(ftplib.error_reply, Exception)
    assert issubclass(ftplib.error_perm, Exception)
    print("exception_hierarchy ok")


def test_exception_catch_as_error():
    try:
        raise ftplib.error_perm("550 File not found")
    except ftplib.Error as e:
        assert "550" in str(e)
    try:
        raise ftplib.error_temp("425 Cannot open data connection")
    except ftplib.Error as e:
        assert "425" in str(e)
    try:
        raise ftplib.error_reply("200 Command OK")
    except ftplib.Error as e:
        assert "200" in str(e)
    try:
        raise ftplib.error_proto("bad response")
    except ftplib.Error as e:
        assert "bad response" in str(e)
    print("exception_catch_as_error ok")


def test_exception_distinct():
    try:
        raise ftplib.error_perm("550 Forbidden")
    except ftplib.error_temp:
        assert False, "error_perm should not be caught as error_temp"
    except ftplib.error_perm:
        pass
    print("exception_distinct ok")


def test_all_errors():
    ae = ftplib.all_errors
    assert isinstance(ae, tuple)
    assert ftplib.Error in ae
    assert OSError in ae
    assert EOFError in ae
    print("all_errors ok")


def test_parse150_with_size():
    r = ftplib.parse150("150 Opening BINARY mode data connection for file.txt (12345 bytes).")
    assert r == 12345, repr(r)
    r2 = ftplib.parse150("150 Opening BINARY mode data connection for file.txt (0 bytes).")
    assert r2 == 0, repr(r2)
    print("parse150_with_size ok")


def test_parse150_no_size():
    r = ftplib.parse150("150 File status okay; about to open data connection.")
    assert r is None, repr(r)
    r2 = ftplib.parse150("150 ASCII mode data connection for /bin/ls.")
    assert r2 is None, repr(r2)
    print("parse150_no_size ok")


def test_parse150_wrong_code():
    try:
        ftplib.parse150("200 Command OK")
        assert False, "should raise error_reply"
    except ftplib.error_reply:
        pass
    print("parse150_wrong_code ok")


def test_parse227_normal():
    host, port = ftplib.parse227("227 Entering Passive Mode (192,168,1,1,4,5)")
    assert host == "192.168.1.1", repr(host)
    assert port == 4 * 256 + 5, repr(port)
    print("parse227_normal ok")


def test_parse227_wrong_code():
    try:
        ftplib.parse227("500 Error")
        assert False, "should raise error_reply"
    except ftplib.error_reply:
        pass
    print("parse227_wrong_code ok")


def test_parse227_bad_format():
    try:
        ftplib.parse227("227 Entering Passive Mode (bad format)")
        assert False, "should raise error_proto"
    except ftplib.error_proto:
        pass
    print("parse227_bad_format ok")


def test_parse229_normal():
    host, port = ftplib.parse229("229 Entering Extended Passive Mode (|||54321|)", ("192.168.1.1", 21))
    assert port == 54321, repr(port)
    assert host == "192.168.1.1", repr(host)
    print("parse229_normal ok")


def test_parse229_wrong_code():
    try:
        ftplib.parse229("500 Error", ("127.0.0.1", 21))
        assert False, "should raise error_reply"
    except ftplib.error_reply:
        pass
    print("parse229_wrong_code ok")


def test_parse257_normal():
    r = ftplib.parse257('257 "/home/user" is the current directory')
    assert r == "/home/user", repr(r)
    r2 = ftplib.parse257('257 "/some/path"')
    assert r2 == "/some/path", repr(r2)
    print("parse257_normal ok")


def test_parse257_non_compliant():
    r = ftplib.parse257("257 /path/without/quotes")
    assert r == "", repr(r)
    print("parse257_non_compliant ok")


def test_parse257_wrong_code():
    try:
        ftplib.parse257("500 Error")
        assert False, "should raise error_reply"
    except ftplib.error_reply:
        pass
    print("parse257_wrong_code ok")


def test_print_line(capsys=None):
    # Just verify it doesn't raise; output goes to stdout
    ftplib.print_line("hello world")
    print("print_line ok")


def test_ftp_default_attrs():
    assert ftplib.FTP.host == ""
    assert ftplib.FTP.port == 21
    assert ftplib.FTP.maxline == 8192
    assert ftplib.FTP.sock is None
    assert ftplib.FTP.file is None
    assert ftplib.FTP.welcome is None
    assert ftplib.FTP.passiveserver == True
    assert ftplib.FTP.trust_server_pasv_ipv4_address == False
    assert ftplib.FTP.debugging == 0
    print("ftp_default_attrs ok")


def test_ftp_construction_no_host():
    ftp = ftplib.FTP()
    assert ftp.host == ""
    assert ftp.port == 21
    assert ftp.debugging == 0
    assert ftp.passiveserver == True
    assert ftp.welcome is None
    print("ftp_construction_no_host ok")


def test_ftp_set_debuglevel():
    ftp = ftplib.FTP()
    assert ftp.debugging == 0
    ftp.set_debuglevel(2)
    assert ftp.debugging == 2
    ftp.set_debuglevel(0)
    assert ftp.debugging == 0
    print("ftp_set_debuglevel ok")


def test_ftp_debug_alias():
    ftp = ftplib.FTP()
    ftp.debug(1)
    assert ftp.debugging == 1
    ftp.debug(0)
    assert ftp.debugging == 0
    print("ftp_debug_alias ok")


def test_ftp_set_pasv():
    ftp = ftplib.FTP()
    assert ftp.passiveserver == True
    ftp.set_pasv(False)
    assert ftp.passiveserver == False
    ftp.set_pasv(True)
    assert ftp.passiveserver == True
    print("ftp_set_pasv ok")


def test_ftp_sanitize_pass():
    ftp = ftplib.FTP()
    r = ftp.sanitize("PASS secret123")
    assert "PASS" in r
    assert "secret123" not in r
    assert "*" in r
    print("ftp_sanitize_pass ok")


def test_ftp_sanitize_non_pass():
    ftp = ftplib.FTP()
    r = ftp.sanitize("USER admin")
    assert "USER" in r
    assert "admin" in r
    r2 = ftp.sanitize("RETR file.txt")
    assert "RETR" in r2
    print("ftp_sanitize_non_pass ok")


def test_ftp_getwelcome():
    ftp = ftplib.FTP()
    assert ftp.getwelcome() is None
    ftp.welcome = "220 Welcome to FTP"
    assert ftp.getwelcome() == "220 Welcome to FTP"
    print("ftp_getwelcome ok")


def test_ftp_context_manager():
    ftp = ftplib.FTP()
    with ftp as f:
        assert f is ftp
    print("ftp_context_manager ok")


def test_ftp_close():
    ftp = ftplib.FTP()
    ftp.close()
    print("ftp_close ok")


def test_ftp_tls_subclass():
    assert issubclass(ftplib.FTP_TLS, ftplib.FTP)
    print("ftp_tls_subclass ok")


def test_ftp_tls_instance():
    ftps = ftplib.FTP_TLS()
    assert isinstance(ftps, ftplib.FTP)
    assert isinstance(ftps, ftplib.FTP_TLS)
    print("ftp_tls_instance ok")


def test_module_exports():
    assert ftplib.Error is not None
    assert ftplib.FTP is not None
    assert ftplib.FTP_TLS is not None
    assert callable(ftplib.parse150)
    assert callable(ftplib.parse227)
    assert callable(ftplib.parse229)
    assert callable(ftplib.parse257)
    assert callable(ftplib.print_line)
    assert isinstance(ftplib.all_errors, tuple)
    print("module_exports ok")


test_constants()
test_exception_hierarchy()
test_exception_catch_as_error()
test_exception_distinct()
test_all_errors()
test_parse150_with_size()
test_parse150_no_size()
test_parse150_wrong_code()
test_parse227_normal()
test_parse227_wrong_code()
test_parse227_bad_format()
test_parse229_normal()
test_parse229_wrong_code()
test_parse257_normal()
test_parse257_non_compliant()
test_parse257_wrong_code()
test_print_line()
test_ftp_default_attrs()
test_ftp_construction_no_host()
test_ftp_set_debuglevel()
test_ftp_debug_alias()
test_ftp_set_pasv()
test_ftp_sanitize_pass()
test_ftp_sanitize_non_pass()
test_ftp_getwelcome()
test_ftp_context_manager()
test_ftp_close()
test_ftp_tls_subclass()
test_ftp_tls_instance()
test_module_exports()
