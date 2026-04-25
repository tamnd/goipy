package vm

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/tamnd/goipy/object"
)

// sslContextState holds Go-level TLS configuration that cannot be expressed as
// plain Python values: certificates, CA pools, ALPN protocols.
// Mutable scalar properties (check_hostname, verify_mode, options,
// minimum_version, maximum_version) are stored directly in inst.Dict so that
// Python's STORE_ATTR works without a custom __setattr__.
type sslContextState struct {
	protocol  int
	alpnProtos []string
	certs     []tls.Certificate
	rootCAs   *x509.CertPool
	clientCAs *x509.CertPool
}

// tlsVersionName converts a tls version uint16 to the Python version string.
func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLSv1"
	case tls.VersionTLS11:
		return "TLSv1.1"
	case tls.VersionTLS12:
		return "TLSv1.2"
	case tls.VersionTLS13:
		return "TLSv1.3"
	default:
		return fmt.Sprintf("TLS(%x)", v)
	}
}

// buildSSL constructs the ssl module.
func (i *Interp) buildSSL() *object.Module {
	m := &object.Module{Name: "ssl", Dict: object.NewDict()}

	// ── exception classes ─────────────────────────────────────────────────

	sslErrCls := &object.Class{
		Name:  "SSLError",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.osErr},
	}
	m.Dict.SetStr("SSLError", sslErrCls)

	makeSSLSub := func(name string) *object.Class {
		cls := &object.Class{
			Name:  name,
			Dict:  object.NewDict(),
			Bases: []*object.Class{sslErrCls},
		}
		m.Dict.SetStr(name, cls)
		return cls
	}

	makeSSLSub("SSLZeroReturnError")
	makeSSLSub("SSLWantReadError")
	makeSSLSub("SSLWantWriteError")
	makeSSLSub("SSLSyscallError")
	makeSSLSub("SSLEOFError")
	sslCertVerCls := makeSSLSub("SSLCertVerificationError")
	m.Dict.SetStr("CertificateError", sslCertVerCls) // alias

	// ── constants ─────────────────────────────────────────────────────────

	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}
	setBool := func(name string, val bool) {
		m.Dict.SetStr(name, object.BoolOf(val))
	}

	setInt("PROTOCOL_TLS", 2)
	setInt("PROTOCOL_TLS_CLIENT", 16)
	setInt("PROTOCOL_TLS_SERVER", 17)
	setInt("PROTOCOL_SSLv23", 2)

	setInt("CERT_NONE", 0)
	setInt("CERT_OPTIONAL", 1)
	setInt("CERT_REQUIRED", 2)

	setInt("OP_NO_SSLv2", 0x01000000)
	setInt("OP_NO_SSLv3", 0x02000000)
	setInt("OP_NO_TLSv1", 0x04000000)
	setInt("OP_NO_TLSv1_1", 0x10000000)
	setInt("OP_NO_TLSv1_2", 0x08000000)
	setInt("OP_NO_TLSv1_3", 0x20000000)
	setInt("OP_NO_COMPRESSION", 0x00020000)
	setInt("OP_ALL", 0x80000BFF)
	setInt("OP_CIPHER_SERVER_PREFERENCE", 0x00400000)
	setInt("OP_SINGLE_DH_USE", 0x00100000)
	setInt("OP_SINGLE_ECDH_USE", 0x00080000)
	setInt("OP_ENABLE_MIDDLEBOX_COMPAT", 0x00020000)

	setInt("VERIFY_DEFAULT", 0)
	setInt("VERIFY_CRL_CHECK_LEAF", 4)
	setInt("VERIFY_CRL_CHECK_CHAIN", 12)
	setInt("VERIFY_X509_STRICT", 32)
	setInt("VERIFY_X509_PARTIAL_CHAIN", 0x80000)

	setBool("HAS_ALPN", true)
	setBool("HAS_SNI", true)
	setBool("HAS_TLSv1_3", true)
	setBool("HAS_NPN", false)
	setBool("HAS_PSK", false)
	setBool("HAS_PHA", false)
	setBool("HAS_ECDH", true)

	m.Dict.SetStr("OPENSSL_VERSION", &object.Str{V: "Go crypto/tls 1.26"})
	setInt("OPENSSL_VERSION_NUMBER", 0x30600020)
	m.Dict.SetStr("OPENSSL_VERSION_INFO", &object.Tuple{V: []object.Object{
		object.NewInt(3), object.NewInt(6), object.NewInt(0), object.NewInt(0), object.NewInt(0),
	}})

	setInt("ALERT_DESCRIPTION_CLOSE_NOTIFY", 0)
	setInt("ALERT_DESCRIPTION_HANDSHAKE_FAILURE", 40)
	setInt("ALERT_DESCRIPTION_CERTIFICATE_EXPIRED", 45)

	// ── TLSVersion enum ───────────────────────────────────────────────────

	tlsVerCls := &object.Class{Name: "TLSVersion", Dict: object.NewDict()}
	makeTLSVer := func(name string, val int) *object.Instance {
		inst := &object.Instance{Class: tlsVerCls, Dict: object.NewDict()}
		inst.Dict.SetStr("value", object.NewInt(int64(val)))
		inst.Dict.SetStr("name", &object.Str{V: name})
		inst.Dict.SetStr("__int__", &object.BuiltinFunc{Name: "__int__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.NewInt(int64(val)), nil
			}})
		inst.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				// called via callInstanceDunder(inst, "__eq__", other) → a=[other]
				if len(a) == 0 {
					return object.False, nil
				}
				if other, ok := a[0].(*object.Instance); ok {
					if v, ok2 := other.Dict.GetStr("value"); ok2 {
						if n, ok3 := toInt64(v); ok3 {
							return object.BoolOf(int(n) == val), nil
						}
					}
				}
				if n, ok := toInt64(a[0]); ok {
					return object.BoolOf(int(n) == val), nil
				}
				return object.False, nil
			}})
		inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.NewInt(int64(val)), nil
			}})
		tlsVerCls.Dict.SetStr(name, inst)
		return inst
	}

	tlsV12 := makeTLSVer("TLSv1_2", 771)
	tlsV13 := makeTLSVer("TLSv1_3", 772)
	tlsV10 := makeTLSVer("TLSv1", 769)
	tlsV11 := makeTLSVer("TLSv1_1", 770)
	_ = tlsV10
	_ = tlsV11

	tlsVerCls.Dict.SetStr("MINIMUM_SUPPORTED", tlsV12)
	tlsVerCls.Dict.SetStr("MAXIMUM_SUPPORTED", tlsV13)
	m.Dict.SetStr("TLSVersion", tlsVerCls)

	tlsVersionFromPy := func(obj object.Object) uint16 {
		if inst, ok := obj.(*object.Instance); ok {
			if v, ok2 := inst.Dict.GetStr("value"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					return uint16(n)
				}
			}
		}
		if n, ok := toInt64(obj); ok {
			return uint16(n)
		}
		return 0
	}

	// ── Purpose enum ──────────────────────────────────────────────────────

	purposeCls := &object.Class{Name: "Purpose", Dict: object.NewDict()}
	makePurpose := func(name string, val int) *object.Instance {
		inst := &object.Instance{Class: purposeCls, Dict: object.NewDict()}
		inst.Dict.SetStr("value", object.NewInt(int64(val)))
		inst.Dict.SetStr("name", &object.Str{V: name})
		inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Str{V: "Purpose." + name}, nil
			}})
		purposeCls.Dict.SetStr(name, inst)
		m.Dict.SetStr(name, inst)
		return inst
	}
	_ = makePurpose("SERVER_AUTH", 1)
	_ = makePurpose("CLIENT_AUTH", 2)
	m.Dict.SetStr("Purpose", purposeCls)

	// ── SSLContext ────────────────────────────────────────────────────────

	ctxCls := &object.Class{Name: "SSLContext", Dict: object.NewDict()}

	makeSSLContext := func(protocol int) *object.Instance {
		st := &sslContextState{protocol: protocol}

		inst := &object.Instance{Class: ctxCls, Dict: object.NewDict()}

		inst.Dict.SetStr("protocol", object.NewInt(int64(protocol)))

		// Scalar properties stored as plain dict values so Python STORE_ATTR
		// (ctx.check_hostname = False) updates them without a custom __setattr__.
		switch protocol {
		case 16: // PROTOCOL_TLS_CLIENT
			inst.Dict.SetStr("check_hostname", object.True)
			inst.Dict.SetStr("verify_mode", object.NewInt(2)) // CERT_REQUIRED
		default:
			inst.Dict.SetStr("check_hostname", object.False)
			inst.Dict.SetStr("verify_mode", object.NewInt(0)) // CERT_NONE
		}
		inst.Dict.SetStr("options", object.NewInt(0x80000BFF))   // OP_ALL
		inst.Dict.SetStr("minimum_version", tlsV12)
		inst.Dict.SetStr("maximum_version", tlsV13)
		inst.Dict.SetStr("post_handshake_auth", object.False)
		inst.Dict.SetStr("keylog_filename", object.None)
		inst.Dict.SetStr("verify_flags", object.NewInt(0))

		inst.Dict.SetStr("load_cert_chain", &object.BuiltinFunc{Name: "load_cert_chain",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "load_cert_chain() requires certfile")
				}
				certfile := ""
				keyfile := ""
				if s, ok := a[0].(*object.Str); ok {
					certfile = s.V
				}
				if len(a) > 1 {
					if s, ok := a[1].(*object.Str); ok {
						keyfile = s.V
					}
				}
				if kw != nil {
					if v, ok := kw.GetStr("certfile"); ok {
						if s, ok2 := v.(*object.Str); ok2 {
							certfile = s.V
						}
					}
					if v, ok := kw.GetStr("keyfile"); ok && v != object.None {
						if s, ok2 := v.(*object.Str); ok2 {
							keyfile = s.V
						}
					}
				}
				if keyfile == "" {
					keyfile = certfile
				}
				cert, err := tls.LoadX509KeyPair(certfile, keyfile)
				if err != nil {
					return nil, object.Errorf(sslErrCls, "load_cert_chain: %v", err)
				}
				st.certs = append(st.certs, cert)
				return object.None, nil
			}})

		inst.Dict.SetStr("load_verify_locations", &object.BuiltinFunc{Name: "load_verify_locations",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				cafile := ""
				if len(a) > 0 {
					if s, ok := a[0].(*object.Str); ok {
						cafile = s.V
					}
				}
				if kw != nil {
					if v, ok := kw.GetStr("cafile"); ok && v != object.None {
						if s, ok2 := v.(*object.Str); ok2 {
							cafile = s.V
						}
					}
					if v, ok := kw.GetStr("cadata"); ok && v != object.None {
						pemStr := ""
						if s, ok2 := v.(*object.Str); ok2 {
							pemStr = s.V
						}
						if pemStr != "" {
							pool := x509.NewCertPool()
							pool.AppendCertsFromPEM([]byte(pemStr))
							st.rootCAs = pool
						}
						return object.None, nil
					}
				}
				if cafile != "" {
					pemData, err := os.ReadFile(cafile)
					if err != nil {
						return nil, object.Errorf(sslErrCls, "load_verify_locations: %v", err)
					}
					pool := x509.NewCertPool()
					pool.AppendCertsFromPEM(pemData)
					st.rootCAs = pool
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("load_default_certs", &object.BuiltinFunc{Name: "load_default_certs",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				pool, err := x509.SystemCertPool()
				if err == nil {
					st.rootCAs = pool
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("set_default_verify_paths", &object.BuiltinFunc{Name: "set_default_verify_paths",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				pool, err := x509.SystemCertPool()
				if err == nil {
					st.rootCAs = pool
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("set_ciphers", &object.BuiltinFunc{Name: "set_ciphers",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})

		inst.Dict.SetStr("get_ciphers", &object.BuiltinFunc{Name: "get_ciphers",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.List{V: nil}, nil
			}})

		inst.Dict.SetStr("set_alpn_protocols", &object.BuiltinFunc{Name: "set_alpn_protocols",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "set_alpn_protocols() requires protocols list")
				}
				var protos []string
				if lst, ok := a[0].(*object.List); ok {
					for _, v := range lst.V {
						if s, ok2 := v.(*object.Str); ok2 {
							protos = append(protos, s.V)
						}
					}
				}
				st.alpnProtos = protos
				return object.None, nil
			}})

		inst.Dict.SetStr("set_servername_callback", &object.BuiltinFunc{Name: "set_servername_callback",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})

		inst.Dict.SetStr("set_npn_protocols", &object.BuiltinFunc{Name: "set_npn_protocols",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})

		inst.Dict.SetStr("load_dh_params", &object.BuiltinFunc{Name: "load_dh_params",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})

		inst.Dict.SetStr("set_ecdh_curve", &object.BuiltinFunc{Name: "set_ecdh_curve",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})

		inst.Dict.SetStr("wrap_socket", &object.BuiltinFunc{Name: "wrap_socket",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				// inst.Dict method: self is NOT prepended; a[0] is the socket arg.
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "wrap_socket() requires a socket")
				}
				sockObj := a[0]
				serverSide := false
				serverHostname := ""

				if len(a) > 1 {
					serverSide = object.Truthy(a[1])
				}
				if len(a) > 2 {
					if s, ok := a[2].(*object.Str); ok {
						serverHostname = s.V
					}
				}
				if kw != nil {
					if v, ok := kw.GetStr("server_side"); ok {
						serverSide = object.Truthy(v)
					}
					if v, ok := kw.GetStr("server_hostname"); ok && v != object.None {
						if s, ok2 := v.(*object.Str); ok2 {
							serverHostname = s.V
						}
					}
				}

				// Read current scalar property values from inst.Dict (may have been
				// updated by Python's STORE_ATTR since the context was created).
				checkHostname := false
				if v, ok2 := inst.Dict.GetStr("check_hostname"); ok2 {
					checkHostname = object.Truthy(v)
				}
				verifyMode := 0
				if v, ok2 := inst.Dict.GetStr("verify_mode"); ok2 {
					if n, ok3 := toInt64(v); ok3 {
						verifyMode = int(n)
					}
				}
				minVer := uint16(tls.VersionTLS12)
				if v, ok2 := inst.Dict.GetStr("minimum_version"); ok2 {
					if n := tlsVersionFromPy(v); n != 0 {
						minVer = n
					}
				}
				maxVer := uint16(tls.VersionTLS13)
				if v, ok2 := inst.Dict.GetStr("maximum_version"); ok2 {
					if n := tlsVersionFromPy(v); n != 0 {
						maxVer = n
					}
				}

				sockSt := sockStateOf(sockObj)
				if sockSt == nil {
					return nil, object.Errorf(sslErrCls, "wrap_socket() argument is not a socket")
				}
				sockSt.mu.RLock()
				conn := sockSt.conn
				sockSt.mu.RUnlock()
				if conn == nil {
					return nil, object.Errorf(sslErrCls, "socket is not connected")
				}

				cfg := &tls.Config{
					MinVersion:   minVer,
					MaxVersion:   maxVer,
					NextProtos:   append([]string(nil), st.alpnProtos...),
					Certificates: append([]tls.Certificate(nil), st.certs...),
				}
				if serverSide {
					cfg.ClientCAs = st.clientCAs
					if verifyMode == 2 {
						cfg.ClientAuth = tls.RequireAndVerifyClientCert
					} else if verifyMode == 1 {
						cfg.ClientAuth = tls.VerifyClientCertIfGiven
					} else {
						cfg.ClientAuth = tls.NoClientCert
					}
				} else {
					cfg.RootCAs = st.rootCAs
					if verifyMode == 0 || !checkHostname {
						cfg.InsecureSkipVerify = true
					}
					if serverHostname != "" && checkHostname {
						cfg.ServerName = serverHostname
					}
				}

				var tlsConn *tls.Conn
				if serverSide {
					tlsConn = tls.Server(conn, cfg)
				} else {
					tlsConn = tls.Client(conn, cfg)
				}
				if err := tlsConn.Handshake(); err != nil {
					tlsConn.Close() //nolint
					return nil, object.Errorf(sslErrCls, "TLS handshake: %v", err)
				}

				return i.makeSSLSocketInst(tlsConn, inst, serverSide, serverHostname), nil
			}})

		inst.Dict.SetStr("wrap_bio", &object.BuiltinFunc{Name: "wrap_bio",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return nil, object.Errorf(sslErrCls, "wrap_bio() not supported in goipy")
			}})

		return inst
	}

	// ── SSLContext constructor ────────────────────────────────────────────

	m.Dict.SetStr("SSLContext", &object.BuiltinFunc{Name: "SSLContext",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			protocol := 2 // PROTOCOL_TLS
			if len(a) > 0 {
				if n, ok := toInt64(a[0]); ok {
					protocol = int(n)
				}
			}
			return makeSSLContext(protocol), nil
		}})

	// ── create_default_context ────────────────────────────────────────────

	m.Dict.SetStr("create_default_context", &object.BuiltinFunc{Name: "create_default_context",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ctx := makeSSLContext(16) // PROTOCOL_TLS_CLIENT: check_hostname=True, verify_mode=CERT_REQUIRED
			// Load system CA certs
			if fn, ok := ctx.Dict.GetStr("load_default_certs"); ok {
				i.callObject(fn, []object.Object{ctx}, nil) //nolint
			}
			// Handle cafile kwarg
			if kw != nil {
				if cafile, ok := kw.GetStr("cafile"); ok && cafile != object.None {
					if fn, ok2 := ctx.Dict.GetStr("load_verify_locations"); ok2 {
						i.callObject(fn, []object.Object{ctx, cafile}, nil) //nolint
					}
				}
			}
			return ctx, nil
		}})

	// ── get_server_certificate ────────────────────────────────────────────

	m.Dict.SetStr("get_server_certificate", &object.BuiltinFunc{Name: "get_server_certificate",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "get_server_certificate() requires address")
			}
			addrStr, err := parseAddr(a[0])
			if err != nil {
				return nil, object.Errorf(sslErrCls, "%v", err)
			}
			cfg := &tls.Config{InsecureSkipVerify: true}
			conn, err := tls.Dial("tcp", addrStr, cfg)
			if err != nil {
				return nil, object.Errorf(sslErrCls, "get_server_certificate: %v", err)
			}
			certs := conn.ConnectionState().PeerCertificates
			conn.Close() //nolint
			if len(certs) == 0 {
				return nil, object.Errorf(sslErrCls, "no certificate received")
			}
			pemData := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: certs[0].Raw,
			})
			return &object.Str{V: string(pemData)}, nil
		}})

	// ── get_default_verify_paths ──────────────────────────────────────────

	pathsCls := &object.Class{Name: "DefaultVerifyPaths", Dict: object.NewDict()}
	pathsInst := &object.Instance{Class: pathsCls, Dict: object.NewDict()}
	pathsInst.Dict.SetStr("cafile", object.None)
	pathsInst.Dict.SetStr("capath", object.None)
	pathsInst.Dict.SetStr("openssl_cafile_env", &object.Str{V: "SSL_CERT_FILE"})
	pathsInst.Dict.SetStr("openssl_cafile", &object.Str{V: ""})
	pathsInst.Dict.SetStr("openssl_capath_env", &object.Str{V: "SSL_CERT_DIR"})
	pathsInst.Dict.SetStr("openssl_capath", &object.Str{V: ""})

	m.Dict.SetStr("get_default_verify_paths", &object.BuiltinFunc{Name: "get_default_verify_paths",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return pathsInst, nil
		}})

	// ── DER / PEM conversion ──────────────────────────────────────────────

	m.Dict.SetStr("DER_cert_to_PEM_cert", &object.BuiltinFunc{Name: "DER_cert_to_PEM_cert",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "DER_cert_to_PEM_cert() requires bytes")
			}
			b, ok := a[0].(*object.Bytes)
			if !ok {
				return nil, object.Errorf(i.typeErr, "DER_cert_to_PEM_cert() argument must be bytes")
			}
			pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: b.V})
			return &object.Str{V: string(pemData)}, nil
		}})

	m.Dict.SetStr("PEM_cert_to_DER_cert", &object.BuiltinFunc{Name: "PEM_cert_to_DER_cert",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "PEM_cert_to_DER_cert() requires string")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "PEM_cert_to_DER_cert() argument must be str")
			}
			block, _ := pem.Decode([]byte(s.V))
			if block == nil {
				return nil, object.Errorf(sslErrCls, "PEM_cert_to_DER_cert: no PEM data found")
			}
			return &object.Bytes{V: block.Bytes}, nil
		}})

	// ── cert_time_to_seconds ──────────────────────────────────────────────

	m.Dict.SetStr("cert_time_to_seconds", &object.BuiltinFunc{Name: "cert_time_to_seconds",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "cert_time_to_seconds() requires time string")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "cert_time_to_seconds() argument must be str")
			}
			for _, layout := range []string{
				"Jan  2 15:04:05 2006 GMT",
				"Jan 2 15:04:05 2006 GMT",
				"Jan  2 15:04:05 2006 MST",
			} {
				t, err := time.Parse(layout, s.V)
				if err == nil {
					return object.NewInt(t.Unix()), nil
				}
			}
			return object.NewInt(0), nil
		}})

	// ── RAND functions ────────────────────────────────────────────────────

	m.Dict.SetStr("RAND_bytes", &object.BuiltinFunc{Name: "RAND_bytes",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n := int64(16)
			if len(a) > 0 {
				if nn, ok := toInt64(a[0]); ok {
					n = nn
				}
			}
			buf := make([]byte, n)
			if _, err := rand.Read(buf); err != nil {
				return nil, object.Errorf(sslErrCls, "RAND_bytes: %v", err)
			}
			return &object.Bytes{V: buf}, nil
		}})

	m.Dict.SetStr("RAND_status", &object.BuiltinFunc{Name: "RAND_status",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})

	m.Dict.SetStr("RAND_add", &object.BuiltinFunc{Name: "RAND_add",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	// ── enum_certificates / enum_crls (stubs) ────────────────────────────

	m.Dict.SetStr("enum_certificates", &object.BuiltinFunc{Name: "enum_certificates",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: nil}, nil
		}})

	m.Dict.SetStr("enum_crls", &object.BuiltinFunc{Name: "enum_crls",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: nil}, nil
		}})

	// ── MemoryBIO (stub) ──────────────────────────────────────────────────

	bioCls := &object.Class{Name: "MemoryBIO", Dict: object.NewDict()}
	m.Dict.SetStr("MemoryBIO", &object.BuiltinFunc{Name: "MemoryBIO",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			bioInst := &object.Instance{Class: bioCls, Dict: object.NewDict()}
			bioInst.Dict.SetStr("pending", object.NewInt(0))
			bioInst.Dict.SetStr("eof", object.False)
			bioInst.Dict.SetStr("read", &object.BuiltinFunc{Name: "read",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return &object.Bytes{V: nil}, nil
				}})
			bioInst.Dict.SetStr("write", &object.BuiltinFunc{Name: "write",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return object.NewInt(0), nil
				}})
			bioInst.Dict.SetStr("write_eof", &object.BuiltinFunc{Name: "write_eof",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return object.None, nil
				}})
			return bioInst, nil
		}})

	return m
}

// makeSSLSocketInst creates a Python SSLSocket wrapping a *tls.Conn.
func (i *Interp) makeSSLSocketInst(
	tlsConn *tls.Conn,
	ctx object.Object,
	serverSide bool,
	serverHostname string,
) *object.Instance {
	sslCls := &object.Class{Name: "SSLSocket", Dict: object.NewDict()}
	inst := &object.Instance{Class: sslCls, Dict: object.NewDict()}

	inst.Dict.SetStr("context", ctx)
	inst.Dict.SetStr("server_side", object.BoolOf(serverSide))
	if serverSide || serverHostname == "" {
		inst.Dict.SetStr("server_hostname", object.None)
	} else {
		inst.Dict.SetStr("server_hostname", &object.Str{V: serverHostname})
	}

	inst.Dict.SetStr("read", &object.BuiltinFunc{Name: "read",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			n := int64(4096)
			if len(a) > 0 {
				if nn, ok := toInt64(a[0]); ok {
					n = nn
				}
			}
			buf := make([]byte, n)
			nr, err := tlsConn.Read(buf)
			if err != nil && nr == 0 {
				return &object.Bytes{V: []byte{}}, nil
			}
			return &object.Bytes{V: buf[:nr]}, nil
		}})

	inst.Dict.SetStr("write", &object.BuiltinFunc{Name: "write",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "write() requires data")
			}
			b, ok := a[0].(*object.Bytes)
			if !ok {
				return nil, object.Errorf(i.typeErr, "write() argument must be bytes")
			}
			n, err := tlsConn.Write(b.V)
			if err != nil {
				return nil, object.Errorf(i.osErr, "write: %v", err)
			}
			return object.NewInt(int64(n)), nil
		}})

	inst.Dict.SetStr("recv", &object.BuiltinFunc{Name: "recv",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			n := int64(4096)
			if len(a) > 0 {
				if nn, ok := toInt64(a[0]); ok {
					n = nn
				}
			}
			buf := make([]byte, n)
			nr, err := tlsConn.Read(buf)
			if err != nil && nr == 0 {
				return &object.Bytes{V: []byte{}}, nil
			}
			return &object.Bytes{V: buf[:nr]}, nil
		}})

	inst.Dict.SetStr("send", &object.BuiltinFunc{Name: "send",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "send() requires data")
			}
			b, ok := a[0].(*object.Bytes)
			if !ok {
				return nil, object.Errorf(i.typeErr, "send() argument must be bytes")
			}
			n, err := tlsConn.Write(b.V)
			if err != nil {
				return nil, object.Errorf(i.osErr, "send: %v", err)
			}
			return object.NewInt(int64(n)), nil
		}})

	inst.Dict.SetStr("sendall", &object.BuiltinFunc{Name: "sendall",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "sendall() requires data")
			}
			b, ok := a[0].(*object.Bytes)
			if !ok {
				return nil, object.Errorf(i.typeErr, "sendall() argument must be bytes")
			}
			buf := b.V
			for len(buf) > 0 {
				n, err := tlsConn.Write(buf)
				if err != nil {
					return nil, object.Errorf(i.osErr, "sendall: %v", err)
				}
				buf = buf[n:]
			}
			return object.None, nil
		}})

	inst.Dict.SetStr("recvfrom", &object.BuiltinFunc{Name: "recvfrom",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			n := int64(4096)
			if len(a) > 0 {
				if nn, ok := toInt64(a[0]); ok {
					n = nn
				}
			}
			buf := make([]byte, n)
			nr, err := tlsConn.Read(buf)
			if err != nil && nr == 0 {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			addr := netAddrToTuple(tlsConn.RemoteAddr())
			return &object.Tuple{V: []object.Object{&object.Bytes{V: buf[:nr]}, addr}}, nil
		}})

	inst.Dict.SetStr("do_handshake", &object.BuiltinFunc{Name: "do_handshake",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if err := tlsConn.Handshake(); err != nil {
				return nil, object.Errorf(i.osErr, "do_handshake: %v", err)
			}
			return object.None, nil
		}})

	inst.Dict.SetStr("getpeercert", &object.BuiltinFunc{Name: "getpeercert",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			binaryForm := false
			if len(a) > 0 {
				binaryForm = object.Truthy(a[0])
			}
			if kw != nil {
				if v, ok := kw.GetStr("binary_form"); ok {
					binaryForm = object.Truthy(v)
				}
			}
			cs := tlsConn.ConnectionState()
			if len(cs.PeerCertificates) == 0 {
				return object.None, nil
			}
			cert := cs.PeerCertificates[0]
			if binaryForm {
				return &object.Bytes{V: cert.Raw}, nil
			}
			return object.NewDict(), nil
		}})

	inst.Dict.SetStr("get_verified_chain", &object.BuiltinFunc{Name: "get_verified_chain",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: nil}, nil
		}})

	inst.Dict.SetStr("cipher", &object.BuiltinFunc{Name: "cipher",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			cs := tlsConn.ConnectionState()
			if cs.CipherSuite == 0 {
				return object.None, nil
			}
			name := tls.CipherSuiteName(cs.CipherSuite)
			proto := tlsVersionName(cs.Version)
			bits := 256
			if strings.Contains(name, "128") {
				bits = 128
			}
			return &object.Tuple{V: []object.Object{
				&object.Str{V: name},
				&object.Str{V: proto},
				object.NewInt(int64(bits)),
			}}, nil
		}})

	inst.Dict.SetStr("shared_ciphers", &object.BuiltinFunc{Name: "shared_ciphers",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: nil}, nil
		}})

	inst.Dict.SetStr("version", &object.BuiltinFunc{Name: "version",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			cs := tlsConn.ConnectionState()
			if cs.Version == 0 {
				return object.None, nil
			}
			return &object.Str{V: tlsVersionName(cs.Version)}, nil
		}})

	inst.Dict.SetStr("selected_alpn_protocol", &object.BuiltinFunc{Name: "selected_alpn_protocol",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			cs := tlsConn.ConnectionState()
			if cs.NegotiatedProtocol == "" {
				return object.None, nil
			}
			return &object.Str{V: cs.NegotiatedProtocol}, nil
		}})

	inst.Dict.SetStr("selected_npn_protocol", &object.BuiltinFunc{Name: "selected_npn_protocol",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	inst.Dict.SetStr("compression", &object.BuiltinFunc{Name: "compression",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	inst.Dict.SetStr("pending", &object.BuiltinFunc{Name: "pending",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(0), nil
		}})

	inst.Dict.SetStr("get_channel_binding", &object.BuiltinFunc{Name: "get_channel_binding",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	inst.Dict.SetStr("unwrap", &object.BuiltinFunc{Name: "unwrap",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	inst.Dict.SetStr("verify_client_post_handshake", &object.BuiltinFunc{Name: "verify_client_post_handshake",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	inst.Dict.SetStr("getsockname", &object.BuiltinFunc{Name: "getsockname",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return netAddrToTuple(tlsConn.LocalAddr()), nil
		}})

	inst.Dict.SetStr("getpeername", &object.BuiltinFunc{Name: "getpeername",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return netAddrToTuple(tlsConn.RemoteAddr()), nil
		}})

	inst.Dict.SetStr("fileno", &object.BuiltinFunc{Name: "fileno",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(-1), nil
		}})

	inst.Dict.SetStr("setblocking", &object.BuiltinFunc{Name: "setblocking",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	inst.Dict.SetStr("settimeout", &object.BuiltinFunc{Name: "settimeout",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) > 0 && a[0] != object.None {
				if f, ok := toFloat64(a[0]); ok && f > 0 {
					tlsConn.SetDeadline(time.Now().Add( //nolint
						time.Duration(f * float64(time.Second))))
				}
			}
			return object.None, nil
		}})

	inst.Dict.SetStr("gettimeout", &object.BuiltinFunc{Name: "gettimeout",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	inst.Dict.SetStr("getblocking", &object.BuiltinFunc{Name: "getblocking",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})

	inst.Dict.SetStr("setsockopt", &object.BuiltinFunc{Name: "setsockopt",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	inst.Dict.SetStr("getsockopt", &object.BuiltinFunc{Name: "getsockopt",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(0), nil
		}})

	inst.Dict.SetStr("shutdown", &object.BuiltinFunc{Name: "shutdown",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			tlsConn.Close() //nolint
			return object.None, nil
		}})

	inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			tlsConn.Close() //nolint
			return object.None, nil
		}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			tlsConn.Close() //nolint
			return object.False, nil
		}})

	if local := tlsConn.LocalAddr(); local != nil {
		switch local.Network() {
		case "tcp", "tcp4":
			inst.Dict.SetStr("family", object.NewInt(sslAFINET))
			inst.Dict.SetStr("type", object.NewInt(1))
		case "tcp6":
			inst.Dict.SetStr("family", object.NewInt(sslAFINET6))
			inst.Dict.SetStr("type", object.NewInt(1))
		}
	}
	inst.Dict.SetStr("proto", object.NewInt(0))
	inst.Dict.SetStr("session", object.None)
	inst.Dict.SetStr("session_reused", object.False)

	return inst
}

// sslAFINET / sslAFINET6 mirror the syscall constants without importing syscall.
const (
	sslAFINET  = 2
	sslAFINET6 = 30 // macOS value; Linux is 10
)

// Suppress "imported and not used" for packages only needed via side effects.
var _ net.Conn  // ensures net package is imported
var _ = strings.Contains
var _ = time.Now
var _ = fmt.Sprintf
var _ = os.ReadFile
var _ = pem.Block{}
var _ = (*x509.Certificate)(nil)
var _ = tls.Config{}
