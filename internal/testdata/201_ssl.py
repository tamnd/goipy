"""Tests for ssl module."""
import ssl
import socket
import threading
import tempfile
import os

# Self-signed cert: RSA 2048, CN=localhost, SAN=127.0.0.1,DNS:localhost, 10yr
CERT_PEM = """\
-----BEGIN CERTIFICATE-----
MIIDJTCCAg2gAwIBAgIUKKnw2U3oBZkPRc8+EUuLna/od94wDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTI2MDQyNTAzMzgwNloXDTM2MDQy
MjAzMzgwNlowFDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEAqty0Mg4AZqgUNSTYrehFviR4weMSiKxfv1orRzzlZjId
iyg9+vKTk2Eopb3qH3GbA20i7WIBNEZAG89EoaQ4FY/a8PBu/pyBereRlrU9IVt2
Pgk73TTbLQGE0FKpcNIyj8V+0LYL1KzdsY/U995MVSo0kEqXFc546ez1iy06MBcg
oxKg+tKKuMI4O4CFoWRR3vxnXs1wasXvGDqSv3Pt8EAkLYeydOb8NCU8cVFfcGrR
kXG0RymrsXVJJ2ru5BFydz8vrlDosebqJ3wUs3FbdDm67YIrcrS9ZjmeNDZl2Zjt
kw/zZwTcJYSd6+dUQUCPq5u5jZcRGXIofrSGRLU3NQIDAQABo28wbTAdBgNVHQ4E
FgQUa4Qn7Red1dsS5pMXCooATDkwEAIwHwYDVR0jBBgwFoAUa4Qn7Red1dsS5pMX
CooATDkwEAIwDwYDVR0TAQH/BAUwAwEB/zAaBgNVHREEEzARhwR/AAABgglsb2Nh
bGhvc3QwDQYJKoZIhvcNAQELBQADggEBADOQnHgajBfNmPQVh29qY3s8CYqVhPAA
mxfBidXimuStsTgWnIGVr3odSPSIKL2VLqUTrrZ/aVnP4FuevIs3GCNHk2YTMfTN
peCYet5Z3taBrSUu5IjvPUnxznCBD4zE/lR7K6SwyVq+/Ytxx1eOZDKAxIjUhZer
EOC5nhc2yvO20Qm65WpvwZ5mNb4hqufs1nmDlPZ6/M+b+9r+6vMI6rknbAfKWsF/
H9kr7DoVzuUdXeoPFPXGScw79FZ5OoX8850V6/EVhZWaQp3iSbCdavL3114xcxTL
+z5CivZdXwNjz7Ovxyj+kqkj/2CguKMHZgO9lPZ92tm4LMWAmoFkRfM=
-----END CERTIFICATE-----
"""

KEY_PEM = """\
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCq3LQyDgBmqBQ1
JNit6EW+JHjB4xKIrF+/WitHPOVmMh2LKD368pOTYSilveofcZsDbSLtYgE0RkAb
z0ShpDgVj9rw8G7+nIF6t5GWtT0hW3Y+CTvdNNstAYTQUqlw0jKPxX7QtgvUrN2x
j9T33kxVKjSQSpcVznjp7PWLLTowFyCjEqD60oq4wjg7gIWhZFHe/GdezXBqxe8Y
OpK/c+3wQCQth7J05vw0JTxxUV9watGRcbRHKauxdUknau7kEXJ3Py+uUOix5uon
fBSzcVt0ObrtgitytL1mOZ40NmXZmO2TD/NnBNwlhJ3r51RBQI+rm7mNlxEZcih+
tIZEtTc1AgMBAAECggEACosL43rRs4Pzm0PmyvRmOVFu0if84MoiLmCWAxNS5Hd7
MzcMfAuz54g7Rd7uL3qHIjL90MAXau5azlx/06mafFogHHX5o2Rs6PGi3jXYy9Ik
/bb8Jq44SBKr617ubbPKwuxg4ugY2sq/81D4x0LEWwz7qVuw7vuKooSosincSVNK
FZOpQbqa03jyqtEWuo5BkilyFGU6QUw4BAm5d14GaFtvIpRU7SSMHjrOmg9Sgn2Y
fJCdXCu9i12PcR46y3CgALGL93InemRZiHlwNXjpg3+uq+rdRsIWDZh7sX0iPGN6
iiAlgQKIJYg8dJ2e/Ce2BQywdZm7uU4JY4PokHHbkQKBgQDmcIodB6vlofX6FrM1
UPk/0K8fPFAtW5xGhxRl7Tz3aRbotud6mtwQiq9WYnUQkmKMtq5v5gYDLOx+Q8js
flFQpqWF3iZ851JJcqSB1c3JkfzFK8yaF13RcbuSuaNa3a0Ycyi49Z7XvhG+UYoG
3L6kcA24irsSj//2B4NG2aE+lwKBgQC90G2/ntbY3da+CDRGNUMN+lw0xhwc6svi
ykP1jR6Kws2ZyPRRFHEjsyaXAA1lSeUjCkwiZiJImyCLKargnT3DB3MiBvqhsQRM
1xdXLRPO4ejBXdhvk5fyTjkWv6ZN/VT8BKOkuEvc5ggk7gwLspSE4sISFbYudJIv
VCqou7M+EwKBgQDcSAg+5+5yfPheMQTumpEpZ5uACG+8bC5fXREqvjXhbBvyKaXt
lct4JJqnwUaWqNh8GsV1QjXNI0yPBs0zBS1GI4dnCI1SKO6IN4b6dh9Z6Kvw74WO
Z3fPlyDviXWWjwHAtZTf+iv1oEPl4pxHIw06s8Lp/fQGMpFD5rqfJ7rz0QKBgG2Y
esW5ILP24pa2hfhDqIPZmoOeH31S1IYN409UO95CvBOfQ/HMq8bBGfb1wMZK9FJX
D76M2h6x8whm9pLaN170XjW3cd8KJkF3r7JWBKnfJlT/qohB3+m34J6R6aP+MaOR
yirBfov9BZbNG4dlhPt1Mjje8GHCTOWm4zqppcShAoGBAMfiT5PRqT+vIp/4hidx
cUfIfPZgTvIkTBoN7MyzRsMwbiosBr1kL6rnGMCTI7LFacFkFst8kznvcjxD9DCd
MfNdDBkdMsnIsqNwB4oIgPp28W/nF7U8ls0mCk7Z4Zvy3mVsPEvi80YXOo7mxzrP
1CNpEu0Dd/zBlnb5FfIGIT92
-----END PRIVATE KEY-----
"""


def _write_certs():
    d = tempfile.mkdtemp()
    certfile = os.path.join(d, 'cert.pem')
    keyfile = os.path.join(d, 'key.pem')
    with open(certfile, 'w') as f:
        f.write(CERT_PEM)
    with open(keyfile, 'w') as f:
        f.write(KEY_PEM)
    return certfile, keyfile


# ─── constants ───────────────────────────────────────────────────────────────

def test_constants():
    assert ssl.PROTOCOL_TLS_CLIENT == 16
    assert ssl.PROTOCOL_TLS_SERVER == 17
    assert ssl.CERT_NONE == 0
    assert ssl.CERT_OPTIONAL == 1
    assert ssl.CERT_REQUIRED == 2
    assert ssl.TLSVersion.TLSv1_2.value == 771
    assert ssl.TLSVersion.TLSv1_3.value == 772
    assert ssl.HAS_ALPN == True
    assert ssl.HAS_SNI == True
    assert ssl.HAS_TLSv1_3 == True
    assert isinstance(ssl.OPENSSL_VERSION, str)
    assert isinstance(ssl.OPENSSL_VERSION_NUMBER, int)
    assert ssl.OP_NO_SSLv2 is not None
    assert ssl.OP_NO_TLSv1 is not None
    assert ssl.OP_NO_COMPRESSION is not None
    print("constants ok")

# ─── exceptions ──────────────────────────────────────────────────────────────

def test_exceptions():
    assert issubclass(ssl.SSLError, OSError)
    assert issubclass(ssl.SSLZeroReturnError, ssl.SSLError)
    assert issubclass(ssl.SSLWantReadError, ssl.SSLError)
    assert issubclass(ssl.SSLWantWriteError, ssl.SSLError)
    assert issubclass(ssl.SSLEOFError, ssl.SSLError)
    assert issubclass(ssl.SSLCertVerificationError, ssl.SSLError)
    assert issubclass(ssl.CertificateError, ssl.SSLError)
    print("exceptions ok")

# ─── SSLContext creation ──────────────────────────────────────────────────────

def test_context_creation():
    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
    assert ctx is not None
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    assert ctx.verify_mode == ssl.CERT_NONE
    assert ctx.check_hostname == False

    ctx2 = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    assert ctx2 is not None
    assert ctx2.check_hostname == False
    print("context creation ok")

# ─── create_default_context ──────────────────────────────────────────────────

def test_create_default_context():
    ctx = ssl.create_default_context()
    assert ctx is not None
    assert ctx.verify_mode == ssl.CERT_REQUIRED
    # Can change settings
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    print("create_default_context ok")

# ─── SSLContext options ───────────────────────────────────────────────────────

def test_context_options():
    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    ctx.set_ciphers('DEFAULT')
    ctx.set_alpn_protocols(['h2', 'http/1.1'])
    ctx.minimum_version = ssl.TLSVersion.TLSv1_2
    ctx.maximum_version = ssl.TLSVersion.TLSv1_3
    assert ctx.minimum_version == ssl.TLSVersion.TLSv1_2
    assert ctx.maximum_version == ssl.TLSVersion.TLSv1_3
    print("context options ok")

# ─── load_cert_chain ─────────────────────────────────────────────────────────

def test_load_cert_chain():
    certfile, keyfile = _write_certs()
    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    ctx.load_cert_chain(certfile, keyfile)
    print("load_cert_chain ok")

# ─── get_default_verify_paths ────────────────────────────────────────────────

def test_default_verify_paths():
    paths = ssl.get_default_verify_paths()
    assert hasattr(paths, 'cafile') or hasattr(paths, 'openssl_cafile')
    print("default_verify_paths ok")

# ─── DER / PEM conversion ────────────────────────────────────────────────────

def test_der_pem():
    der = ssl.PEM_cert_to_DER_cert(CERT_PEM)
    assert isinstance(der, bytes) and len(der) > 0
    pem = ssl.DER_cert_to_PEM_cert(der)
    assert isinstance(pem, str) and '-----BEGIN CERTIFICATE-----' in pem
    print("der/pem ok")

# ─── TLS loopback ────────────────────────────────────────────────────────────

def test_tls_loopback():
    certfile, keyfile = _write_certs()

    server_ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    server_ctx.load_cert_chain(certfile, keyfile)

    client_ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
    client_ctx.check_hostname = False
    client_ctx.verify_mode = ssl.CERT_NONE

    raw_server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    raw_server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    raw_server.bind(('127.0.0.1', 0))
    port = raw_server.getsockname()[1]
    raw_server.listen(1)

    received = []
    errors = []
    def serve():
        try:
            conn, _ = raw_server.accept()
            ssl_conn = server_ctx.wrap_socket(conn, server_side=True)
            data = ssl_conn.recv(1024)
            received.append(data)
            ssl_conn.sendall(b'tls_pong')
            ssl_conn.close()
        except Exception as e:
            errors.append(str(e))
        finally:
            raw_server.close()

    t = threading.Thread(target=serve)
    t.start()

    raw_client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    raw_client.connect(('127.0.0.1', port))
    ssl_client = client_ctx.wrap_socket(raw_client, server_hostname='localhost')

    ssl_client.sendall(b'tls_ping')
    resp = ssl_client.recv(1024)

    ver = ssl_client.version()
    ciph = ssl_client.cipher()
    cert = ssl_client.getpeercert()

    ssl_client.close()
    t.join()

    assert not errors, f"server error: {errors}"
    assert received[0] == b'tls_ping', f"got {received[0]!r}"
    assert resp == b'tls_pong', f"got {resp!r}"
    assert isinstance(ver, str), f"version should be str, got {ver!r}"
    assert isinstance(ciph, tuple) and len(ciph) == 3, f"cipher={ciph!r}"
    assert isinstance(cert, dict), f"cert={cert!r}"
    print("tls loopback ok")

# ─── SSLSocket attributes ────────────────────────────────────────────────────

def test_ssl_socket_attrs():
    certfile, keyfile = _write_certs()

    server_ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    server_ctx.load_cert_chain(certfile, keyfile)
    server_ctx.set_alpn_protocols(['http/1.1'])

    client_ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
    client_ctx.check_hostname = False
    client_ctx.verify_mode = ssl.CERT_NONE
    client_ctx.set_alpn_protocols(['http/1.1'])

    raw_server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    raw_server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    raw_server.bind(('127.0.0.1', 0))
    port = raw_server.getsockname()[1]
    raw_server.listen(1)

    results = {}
    def serve():
        try:
            conn, _ = raw_server.accept()
            ssl_conn = server_ctx.wrap_socket(conn, server_side=True)
            data = ssl_conn.recv(1024)
            results['server_side'] = ssl_conn.server_side
            results['server_hostname'] = ssl_conn.server_hostname
            ssl_conn.sendall(b'ok')
            ssl_conn.close()
        finally:
            raw_server.close()

    t = threading.Thread(target=serve)
    t.start()

    raw_client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    raw_client.connect(('127.0.0.1', port))
    ssl_client = client_ctx.wrap_socket(raw_client, server_hostname='testhost')
    ssl_client.sendall(b'hi')
    ssl_client.recv(1024)

    alpn = ssl_client.selected_alpn_protocol()
    comp = ssl_client.compression()
    pending = ssl_client.pending()

    ssl_client.close()
    t.join()

    assert results.get('server_side') == True
    assert results.get('server_hostname') is None  # server side has no server_hostname
    assert isinstance(pending, int)
    # alpn may be None if not negotiated or 'http/1.1'
    assert alpn is None or alpn == 'http/1.1', f"alpn={alpn!r}"
    # compression is None (TLS 1.3 doesn't support it)
    assert comp is None
    print("ssl socket attrs ok")

# ─── get_server_certificate ──────────────────────────────────────────────────

def test_get_server_certificate():
    certfile, keyfile = _write_certs()

    server_ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    server_ctx.load_cert_chain(certfile, keyfile)

    raw_server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    raw_server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    raw_server.bind(('127.0.0.1', 0))
    port = raw_server.getsockname()[1]
    raw_server.listen(1)

    def serve():
        try:
            conn, _ = raw_server.accept()
            ssl_conn = server_ctx.wrap_socket(conn, server_side=True)
            # get_server_certificate closes after handshake; recv returns b''
            try:
                ssl_conn.recv(1024)
            except Exception:
                pass
            ssl_conn.close()
        except Exception:
            pass
        finally:
            raw_server.close()

    t = threading.Thread(target=serve)
    t.start()

    pem = ssl.get_server_certificate(('127.0.0.1', port))
    t.join()

    assert isinstance(pem, str), f"expected str, got {type(pem)}"
    assert '-----BEGIN CERTIFICATE-----' in pem
    print("get_server_certificate ok")

# ─── RAND functions ──────────────────────────────────────────────────────────

def test_rand():
    b = ssl.RAND_bytes(16)
    assert isinstance(b, bytes) and len(b) == 16
    status = ssl.RAND_status()
    assert status == True or status == 1
    print("rand ok")

if __name__ == "__main__":
    test_constants()
    test_exceptions()
    test_context_creation()
    test_create_default_context()
    test_context_options()
    test_load_cert_chain()
    test_default_verify_paths()
    test_der_pem()
    test_tls_loopback()
    test_ssl_socket_attrs()
    test_get_server_certificate()
    test_rand()
    print("ALL OK")
