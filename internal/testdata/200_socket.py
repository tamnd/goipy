"""Tests for socket module."""
import socket
import threading

# ─── constants ───────────────────────────────────────────────────────────────

def test_constants():
    assert socket.AF_INET is not None
    assert socket.AF_INET6 is not None
    assert socket.AF_UNIX is not None
    assert socket.SOCK_STREAM is not None
    assert socket.SOCK_DGRAM is not None
    assert socket.IPPROTO_TCP is not None
    assert socket.IPPROTO_UDP is not None
    assert socket.SOL_SOCKET is not None
    assert socket.SO_REUSEADDR is not None
    assert socket.SHUT_RD == 0
    assert socket.SHUT_WR == 1
    assert socket.SHUT_RDWR == 2
    assert socket.INADDR_ANY == 0
    print("constants ok")

# ─── byte order ──────────────────────────────────────────────────────────────

def test_byte_order():
    assert socket.ntohs(socket.htons(1234)) == 1234
    assert socket.ntohl(socket.htonl(1234567)) == 1234567
    assert socket.htons(0) == 0
    assert socket.htonl(0) == 0
    print("byte order ok")

# ─── inet_aton / inet_ntoa ───────────────────────────────────────────────────

def test_inet_aton_ntoa():
    packed = socket.inet_aton('127.0.0.1')
    assert isinstance(packed, bytes) and len(packed) == 4
    assert socket.inet_ntoa(packed) == '127.0.0.1'
    assert socket.inet_aton('0.0.0.0') == b'\x00\x00\x00\x00'
    assert socket.inet_aton('255.255.255.255') == b'\xff\xff\xff\xff'
    print("inet_aton/ntoa ok")

# ─── inet_pton / inet_ntop ───────────────────────────────────────────────────

def test_inet_pton_ntop():
    p4 = socket.inet_pton(socket.AF_INET, '192.168.1.1')
    assert len(p4) == 4
    assert socket.inet_ntop(socket.AF_INET, p4) == '192.168.1.1'
    p6 = socket.inet_pton(socket.AF_INET6, '::1')
    assert len(p6) == 16
    assert socket.inet_ntop(socket.AF_INET6, p6) == '::1'
    print("inet_pton/ntop ok")

# ─── gethostname ─────────────────────────────────────────────────────────────

def test_gethostname():
    h = socket.gethostname()
    assert isinstance(h, str) and len(h) > 0
    print("gethostname ok")

# ─── getaddrinfo ─────────────────────────────────────────────────────────────

def test_getaddrinfo():
    results = socket.getaddrinfo('127.0.0.1', 80)
    assert len(results) > 0
    family, type_, proto, canonname, sockaddr = results[0]
    assert family == socket.AF_INET
    assert sockaddr[0] == '127.0.0.1'
    assert sockaddr[1] == 80
    print("getaddrinfo ok")

# ─── socket creation ─────────────────────────────────────────────────────────

def test_socket_create():
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    assert s.family == socket.AF_INET
    assert s.type == socket.SOCK_STREAM
    s.close()
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    assert s.type == socket.SOCK_DGRAM
    s.close()
    print("socket create ok")

# ─── TCP loopback ────────────────────────────────────────────────────────────

def test_tcp_loopback():
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(('127.0.0.1', 0))
    port = server.getsockname()[1]
    server.listen(1)

    received = []
    def serve():
        conn, addr = server.accept()
        data = conn.recv(1024)
        received.append(data)
        conn.sendall(b'pong')
        conn.close()
        server.close()

    t = threading.Thread(target=serve)
    t.start()

    client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    client.connect(('127.0.0.1', port))
    client.sendall(b'ping')
    resp = client.recv(1024)
    client.close()
    t.join()

    assert received[0] == b'ping', f"got {received[0]!r}"
    assert resp == b'pong', f"got {resp!r}"
    print("tcp loopback ok")

# ─── socketpair ──────────────────────────────────────────────────────────────

def test_socketpair():
    a, b = socket.socketpair()
    a.sendall(b'hello')
    data = b.recv(1024)
    assert data == b'hello', f"got {data!r}"
    b.sendall(b'world')
    data = a.recv(1024)
    assert data == b'world', f"got {data!r}"
    a.close()
    b.close()
    print("socketpair ok")

# ─── UDP ─────────────────────────────────────────────────────────────────────

def test_udp():
    server = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    server.bind(('127.0.0.1', 0))
    port = server.getsockname()[1]
    client = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    client.sendto(b'udp hello', ('127.0.0.1', port))
    data, addr = server.recvfrom(1024)
    assert data == b'udp hello', f"got {data!r}"
    assert addr[0] == '127.0.0.1'
    server.close()
    client.close()
    print("udp ok")

# ─── setblocking / settimeout ─────────────────────────────────────────────────

def test_blocking():
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    assert s.getblocking() == True
    assert s.gettimeout() is None
    s.setblocking(False)
    assert s.getblocking() == False
    assert s.gettimeout() == 0.0
    s.settimeout(5.0)
    assert s.gettimeout() == 5.0
    assert s.getblocking() == True
    s.settimeout(None)
    assert s.gettimeout() is None
    assert s.getblocking() == True
    s.close()
    print("blocking ok")

# ─── setsockopt / getsockopt ─────────────────────────────────────────────────

def test_sockopts():
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    val = s.getsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR)
    assert val != 0, f"expected non-zero, got {val}"
    s.close()
    print("sockopts ok")

# ─── getsockname / getpeername ───────────────────────────────────────────────

def test_socknames():
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(('127.0.0.1', 0))
    port = server.getsockname()[1]
    assert isinstance(port, int) and port > 0
    server.listen(1)

    done = []
    def serve():
        conn, _ = server.accept()
        conn.close()
        server.close()
        done.append(1)

    t = threading.Thread(target=serve)
    t.start()

    client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    client.connect(('127.0.0.1', port))
    peer = client.getpeername()
    assert peer[0] == '127.0.0.1'
    assert peer[1] == port
    local = client.getsockname()
    assert local[0] == '127.0.0.1'
    client.close()
    t.join()
    print("socknames ok")

# ─── error hierarchy ─────────────────────────────────────────────────────────

def test_errors():
    assert issubclass(socket.error, OSError)
    assert issubclass(socket.timeout, socket.error)
    assert issubclass(socket.gaierror, socket.error)
    assert issubclass(socket.herror, socket.error)
    print("errors ok")

# ─── default timeout ─────────────────────────────────────────────────────────

def test_default_timeout():
    socket.setdefaulttimeout(10.0)
    assert socket.getdefaulttimeout() == 10.0
    socket.setdefaulttimeout(None)
    assert socket.getdefaulttimeout() is None
    print("default timeout ok")

# ─── getservbyname / getprotobyname ──────────────────────────────────────────

def test_service_proto():
    p = socket.getprotobyname('tcp')
    assert p == 6, f"expected 6, got {p}"
    p = socket.getprotobyname('udp')
    assert p == 17, f"expected 17, got {p}"
    port = socket.getservbyname('http')
    assert port == 80, f"expected 80, got {port}"
    print("service/proto ok")

# ─── gethostbyname ───────────────────────────────────────────────────────────

def test_gethostbyname():
    ip = socket.gethostbyname('localhost')
    assert ip in ('127.0.0.1', '::1'), f"got {ip!r}"
    ip2 = socket.gethostbyname('127.0.0.1')
    assert ip2 == '127.0.0.1'
    print("gethostbyname ok")

# ─── connect_ex ──────────────────────────────────────────────────────────────

def test_connect_ex():
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(('127.0.0.1', 0))
    port = server.getsockname()[1]
    server.listen(1)

    done = []
    def serve():
        conn, _ = server.accept()
        conn.close()
        server.close()
        done.append(1)

    t = threading.Thread(target=serve)
    t.start()

    client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    err = client.connect_ex(('127.0.0.1', port))
    assert err == 0, f"connect_ex returned {err}"
    client.close()
    t.join()
    print("connect_ex ok")

# ─── create_connection ────────────────────────────────────────────────────────

def test_create_connection():
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(('127.0.0.1', 0))
    port = server.getsockname()[1]
    server.listen(1)

    done = []
    def serve():
        conn, _ = server.accept()
        conn.close()
        server.close()
        done.append(1)

    t = threading.Thread(target=serve)
    t.start()

    conn = socket.create_connection(('127.0.0.1', port))
    assert conn is not None
    conn.close()
    t.join()
    print("create_connection ok")

if __name__ == "__main__":
    test_constants()
    test_byte_order()
    test_inet_aton_ntoa()
    test_inet_pton_ntop()
    test_gethostname()
    test_getaddrinfo()
    test_socket_create()
    test_tcp_loopback()
    test_socketpair()
    test_udp()
    test_blocking()
    test_sockopts()
    test_socknames()
    test_errors()
    test_default_timeout()
    test_service_proto()
    test_gethostbyname()
    test_connect_ex()
    test_create_connection()
    print("ALL OK")
