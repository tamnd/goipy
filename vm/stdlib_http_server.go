package vm

import (
	"github.com/tamnd/goipy/object"
)

const defaultErrorMessage = "<!DOCTYPE HTML>\n<html lang=\"en\">\n    <head>\n        <meta charset=\"utf-8\">\n" +
	"        <style type=\"text/css\">\n            :root {\n                color-scheme: light dark;\n" +
	"            }\n        </style>\n        <title>Error response</title>\n    </head>\n    <body>\n" +
	"        <h1>Error response</h1>\n        <p>Error code: %(code)d</p>\n" +
	"        <p>Message: %(message)s.</p>\n        <p>Error code explanation: %(code)s - %(explain)s.</p>\n" +
	"    </body>\n</html>\n"

func (i *Interp) buildHttpServer() *object.Module {
	m := &object.Module{Name: "http.server", Dict: object.NewDict()}

	// ── Base classes from socketserver ────────────────────────────────────────

	var tcpServerCls, threadingMixInCls, streamHandlerCls *object.Class
	if ssMod, err := i.loadModule("socketserver"); err == nil && ssMod != nil {
		if v, ok := ssMod.Dict.GetStr("TCPServer"); ok {
			tcpServerCls, _ = v.(*object.Class)
		}
		if v, ok := ssMod.Dict.GetStr("ThreadingMixIn"); ok {
			threadingMixInCls, _ = v.(*object.Class)
		}
		if v, ok := ssMod.Dict.GetStr("StreamRequestHandler"); ok {
			streamHandlerCls, _ = v.(*object.Class)
		}
	}
	if tcpServerCls == nil {
		tcpServerCls = &object.Class{Name: "TCPServer", Dict: object.NewDict()}
	}
	if threadingMixInCls == nil {
		threadingMixInCls = &object.Class{Name: "ThreadingMixIn", Dict: object.NewDict()}
	}
	if streamHandlerCls == nil {
		streamHandlerCls = &object.Class{Name: "StreamRequestHandler", Dict: object.NewDict()}
	}

	// ── HTTPMessage from http.client ─────────────────────────────────────────

	var httpMessageCls *object.Class
	if clientMod, err := i.loadModule("http.client"); err == nil && clientMod != nil {
		if v, ok := clientMod.Dict.GetStr("HTTPMessage"); ok {
			httpMessageCls, _ = v.(*object.Class)
		}
	}
	if httpMessageCls == nil {
		httpMessageCls = &object.Class{Name: "HTTPMessage", Dict: object.NewDict()}
	}

	noop := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}
	}

	// ── Module constants ──────────────────────────────────────────────────────

	m.Dict.SetStr("DEFAULT_ERROR_CONTENT_TYPE", &object.Str{V: "text/html;charset=utf-8"})
	m.Dict.SetStr("DEFAULT_ERROR_MESSAGE", &object.Str{V: defaultErrorMessage})

	// ── HTTPServer(TCPServer) ─────────────────────────────────────────────────

	httpServerCls := &object.Class{
		Name:  "HTTPServer",
		Bases: []*object.Class{tcpServerCls},
		Dict:  object.NewDict(),
	}
	httpServerCls.Dict.SetStr("allow_reuse_address", object.True)
	httpServerCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		if len(a) >= 2 {
			inst.Dict.SetStr("server_address", a[1])
		}
		if len(a) >= 3 {
			inst.Dict.SetStr("RequestHandlerClass", a[2])
		}
		return object.None, nil
	}})
	httpServerCls.Dict.SetStr("server_bind", noop("server_bind"))
	httpServerCls.Dict.SetStr("server_close", noop("server_close"))

	m.Dict.SetStr("HTTPServer", httpServerCls)

	// ── ThreadingHTTPServer(ThreadingMixIn, HTTPServer) ───────────────────────

	threadingHttpServerCls := &object.Class{
		Name:  "ThreadingHTTPServer",
		Bases: []*object.Class{threadingMixInCls, httpServerCls},
		Dict:  object.NewDict(),
	}
	m.Dict.SetStr("ThreadingHTTPServer", threadingHttpServerCls)

	// ── responses dict (int key → (phrase, description)) ─────────────────────

	type resp struct {
		code  int64
		phrase string
		desc   string
	}
	allResponses := []resp{
		{100, "Continue", "Request received, please continue"},
		{101, "Switching Protocols", "Switching to new protocol; obey Upgrade header"},
		{102, "Processing", "Server is processing the request"},
		{103, "Early Hints", "Headers sent to prepare for the response"},
		{200, "OK", "Request fulfilled, document follows"},
		{201, "Created", "Document created, URL follows"},
		{202, "Accepted", "Request accepted, processing continues off-line"},
		{203, "Non-Authoritative Information", "Request fulfilled from cache"},
		{204, "No Content", "Request fulfilled, nothing follows"},
		{205, "Reset Content", "Clear input form for further input"},
		{206, "Partial Content", "Partial content follows"},
		{207, "Multi-Status", "Response contains multiple statuses in the body"},
		{208, "Already Reported", "Operation has already been reported"},
		{226, "IM Used", "Request completed using instance manipulations"},
		{300, "Multiple Choices", "Object has several resources -- see URI list"},
		{301, "Moved Permanently", "Object moved permanently -- see URI list"},
		{302, "Found", "Object moved temporarily -- see URI list"},
		{303, "See Other", "Object moved -- see Method and URL list"},
		{304, "Not Modified", "Document has not changed since given time"},
		{305, "Use Proxy", "You must use proxy specified in Location to access this resource"},
		{307, "Temporary Redirect", "Object moved temporarily -- see URI list"},
		{308, "Permanent Redirect", "Object moved permanently -- see URI list"},
		{400, "Bad Request", "Bad request syntax or unsupported method"},
		{401, "Unauthorized", "No permission -- see authorization schemes"},
		{402, "Payment Required", "No payment -- see charging schemes"},
		{403, "Forbidden", "Request forbidden -- authorization will not help"},
		{404, "Not Found", "Nothing matches the given URI"},
		{405, "Method Not Allowed", "Specified method is invalid for this resource"},
		{406, "Not Acceptable", "URI not available in preferred format"},
		{407, "Proxy Authentication Required", "You must authenticate with this proxy before proceeding"},
		{408, "Request Timeout", "Request timed out; try again later"},
		{409, "Conflict", "Request conflict"},
		{410, "Gone", "URI no longer exists and has been permanently removed"},
		{411, "Length Required", "Client must specify Content-Length"},
		{412, "Precondition Failed", "Precondition in headers is false"},
		{413, "Content Too Large", "Content is too large"},
		{414, "URI Too Long", "URI is too long"},
		{415, "Unsupported Media Type", "Entity body in unsupported format"},
		{416, "Range Not Satisfiable", "Cannot satisfy request range"},
		{417, "Expectation Failed", "Expect condition could not be satisfied"},
		{418, "I'm a Teapot", "Server refuses to brew coffee because it is a teapot"},
		{421, "Misdirected Request", "Server is not able to produce a response"},
		{422, "Unprocessable Content", "Server is not able to process the contained instructions"},
		{423, "Locked", "Resource of a method is locked"},
		{424, "Failed Dependency", "Dependent action of the request failed"},
		{425, "Too Early", "Server refuses to process a request that might be replayed"},
		{426, "Upgrade Required", "Server refuses to perform the request using the current protocol"},
		{428, "Precondition Required", "The origin server requires the request to be conditional"},
		{429, "Too Many Requests", "The user has sent too many requests in a given amount of time (\"rate limiting\")"},
		{431, "Request Header Fields Too Large", "The server is unwilling to process the request because its header fields are too large"},
		{451, "Unavailable For Legal Reasons", "The server is denying access to the resource as a consequence of a legal demand"},
		{500, "Internal Server Error", "Server got itself in trouble"},
		{501, "Not Implemented", "Server does not support this operation"},
		{502, "Bad Gateway", "Invalid responses from another server/proxy"},
		{503, "Service Unavailable", "The server cannot process the request due to a high load"},
		{504, "Gateway Timeout", "The gateway server did not receive a timely response"},
		{505, "HTTP Version Not Supported", "Cannot fulfill request"},
		{506, "Variant Also Negotiates", "Server has an internal configuration error"},
		{507, "Insufficient Storage", "Server is not able to store the representation"},
		{508, "Loop Detected", "Server encountered an infinite loop while processing a request"},
		{510, "Not Extended", "Request does not meet the resource access policy"},
		{511, "Network Authentication Required", "The client needs to authenticate to gain network access"},
	}
	responsesDict := object.NewDict()
	for _, r := range allResponses {
		tup := &object.Tuple{V: []object.Object{
			&object.Str{V: r.phrase},
			&object.Str{V: r.desc},
		}}
		_ = responsesDict.Set(object.NewInt(r.code), tup)
	}

	// ── BaseHTTPRequestHandler(StreamRequestHandler) ──────────────────────────

	baseHandlerCls := &object.Class{
		Name:  "BaseHTTPRequestHandler",
		Bases: []*object.Class{streamHandlerCls},
		Dict:  object.NewDict(),
	}
	baseHandlerCls.Dict.SetStr("server_version", &object.Str{V: "BaseHTTP/0.6"})
	baseHandlerCls.Dict.SetStr("sys_version", &object.Str{V: "Python/3.14"})
	baseHandlerCls.Dict.SetStr("protocol_version", &object.Str{V: "HTTP/1.0"})
	baseHandlerCls.Dict.SetStr("default_request_version", &object.Str{V: "HTTP/0.9"})
	baseHandlerCls.Dict.SetStr("error_message_format", &object.Str{V: defaultErrorMessage})
	baseHandlerCls.Dict.SetStr("error_content_type", &object.Str{V: "text/html;charset=utf-8"})
	baseHandlerCls.Dict.SetStr("MessageClass", httpMessageCls)
	baseHandlerCls.Dict.SetStr("disable_nagle_algorithm", object.False)
	baseHandlerCls.Dict.SetStr("rbufsize", object.NewInt(-1))
	baseHandlerCls.Dict.SetStr("wbufsize", object.NewInt(0))
	baseHandlerCls.Dict.SetStr("timeout", object.None)
	baseHandlerCls.Dict.SetStr("responses", responsesDict)

	baseHandlerCls.Dict.SetStr("weekdayname", &object.List{V: []object.Object{
		&object.Str{V: "Mon"}, &object.Str{V: "Tue"}, &object.Str{V: "Wed"},
		&object.Str{V: "Thu"}, &object.Str{V: "Fri"}, &object.Str{V: "Sat"}, &object.Str{V: "Sun"},
	}})
	baseHandlerCls.Dict.SetStr("monthname", &object.List{V: []object.Object{
		object.None,
		&object.Str{V: "Jan"}, &object.Str{V: "Feb"}, &object.Str{V: "Mar"},
		&object.Str{V: "Apr"}, &object.Str{V: "May"}, &object.Str{V: "Jun"},
		&object.Str{V: "Jul"}, &object.Str{V: "Aug"}, &object.Str{V: "Sep"},
		&object.Str{V: "Oct"}, &object.Str{V: "Nov"}, &object.Str{V: "Dec"},
	}})

	for _, name := range []string{
		"handle", "handle_one_request", "handle_expect_100",
		"send_response", "send_response_only", "send_header",
		"end_headers", "flush_headers", "send_error",
		"log_request", "log_error", "log_message",
		"version_string", "date_time_string", "log_date_time_string",
		"address_string", "parse_request", "setup", "finish",
	} {
		baseHandlerCls.Dict.SetStr(name, noop(name))
	}

	m.Dict.SetStr("BaseHTTPRequestHandler", baseHandlerCls)

	// ── SimpleHTTPRequestHandler(BaseHTTPRequestHandler) ─────────────────────

	simpleHandlerCls := &object.Class{
		Name:  "SimpleHTTPRequestHandler",
		Bases: []*object.Class{baseHandlerCls},
		Dict:  object.NewDict(),
	}
	simpleHandlerCls.Dict.SetStr("server_version", &object.Str{V: "SimpleHTTP/0.6"})

	extMap := object.NewDict()
	extMap.SetStr(".gz", &object.Str{V: "application/gzip"})
	extMap.SetStr(".Z", &object.Str{V: "application/octet-stream"})
	extMap.SetStr(".bz2", &object.Str{V: "application/x-bzip2"})
	extMap.SetStr(".xz", &object.Str{V: "application/x-xz"})
	simpleHandlerCls.Dict.SetStr("extensions_map", extMap)

	simpleHandlerCls.Dict.SetStr("do_GET", noop("do_GET"))
	simpleHandlerCls.Dict.SetStr("do_HEAD", noop("do_HEAD"))

	m.Dict.SetStr("SimpleHTTPRequestHandler", simpleHandlerCls)

	// ── CGIHTTPRequestHandler(BaseHTTPRequestHandler) ────────────────────────

	cgiHandlerCls := &object.Class{
		Name:  "CGIHTTPRequestHandler",
		Bases: []*object.Class{baseHandlerCls},
		Dict:  object.NewDict(),
	}
	cgiHandlerCls.Dict.SetStr("cgi_directories", &object.List{V: []object.Object{
		&object.Str{V: "/cgi-bin"},
		&object.Str{V: "/htbin"},
	}})
	cgiHandlerCls.Dict.SetStr("do_GET", noop("do_GET"))
	cgiHandlerCls.Dict.SetStr("do_HEAD", noop("do_HEAD"))
	cgiHandlerCls.Dict.SetStr("do_POST", noop("do_POST"))

	m.Dict.SetStr("CGIHTTPRequestHandler", cgiHandlerCls)

	return m
}
