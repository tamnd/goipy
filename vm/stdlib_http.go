package vm

import (
	"fmt"

	"github.com/tamnd/goipy/object"
)

// buildHttp constructs the top-level `http` package with HTTPStatus and
// HTTPMethod enums matching CPython 3.14's http/__init__.py.
func (i *Interp) buildHttp() *object.Module {
	m := &object.Module{Name: "http", Dict: object.NewDict()}
	m.Dict.SetStr("__path__", &object.List{V: []object.Object{&object.Str{V: ""}}})
	m.Dict.SetStr("__package__", &object.Str{V: "http"})

	// ── HTTPStatus ────────────────────────────────────────────────────────────

	httpStatusCls := &object.Class{Name: "HTTPStatus", Dict: object.NewDict()}

	memberName := func(inst *object.Instance) string {
		if v, ok := inst.Dict.GetStr("_name_"); ok {
			if s, ok := v.(*object.Str); ok {
				return s.V
			}
		}
		return "?"
	}
	memberIntVal := func(inst *object.Instance) int64 {
		if v, ok := inst.Dict.GetStr("_value_"); ok {
			if n, ok := toInt64(v); ok {
				return n
			}
		}
		return 0
	}

	httpStatusCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if inst, ok := a[0].(*object.Instance); ok {
			return &object.Str{V: fmt.Sprintf("<HTTPStatus.%s: %d>", memberName(inst), memberIntVal(inst))}, nil
		}
		return &object.Str{V: "<HTTPStatus>"}, nil
	}})
	httpStatusCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if inst, ok := a[0].(*object.Instance); ok {
			return &object.Str{V: fmt.Sprintf("%d", memberIntVal(inst))}, nil
		}
		return &object.Str{V: "0"}, nil
	}})
	httpStatusCls.Dict.SetStr("__int__", &object.BuiltinFunc{Name: "__int__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if inst, ok := a[0].(*object.Instance); ok {
			return object.NewInt(memberIntVal(inst)), nil
		}
		return object.NewInt(0), nil
	}})
	httpStatusCls.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		if a[0] == a[1] {
			return object.True, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.False, nil
		}
		lv := memberIntVal(inst)
		rv, ok2 := toInt64(a[1])
		if !ok2 {
			if inst2, ok3 := a[1].(*object.Instance); ok3 {
				if v2, ok4 := inst2.Dict.GetStr("_value_"); ok4 {
					rv, ok2 = toInt64(v2)
				}
			}
		}
		if !ok2 {
			return object.False, nil
		}
		return &object.Bool{V: lv == rv}, nil
	}})
	httpStatusCls.Dict.SetStr("__ne__", &object.BuiltinFunc{Name: "__ne__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.True, nil
		}
		if a[0] == a[1] {
			return object.False, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.True, nil
		}
		lv := memberIntVal(inst)
		rv, ok2 := toInt64(a[1])
		if !ok2 {
			if inst2, ok3 := a[1].(*object.Instance); ok3 {
				if v2, ok4 := inst2.Dict.GetStr("_value_"); ok4 {
					rv, ok2 = toInt64(v2)
				}
			}
		}
		if !ok2 {
			return object.True, nil
		}
		return &object.Bool{V: lv != rv}, nil
	}})
	httpStatusCls.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if inst, ok := a[0].(*object.Instance); ok {
			return object.NewInt(memberIntVal(inst)), nil
		}
		return object.NewInt(0), nil
	}})
	httpStatusCls.Dict.SetStr("__lt__", &object.BuiltinFunc{Name: "__lt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.NotImplemented, nil
		}
		lv, rv, ok := httpStatusCmpVals(a)
		if !ok {
			return object.NotImplemented, nil
		}
		return &object.Bool{V: lv < rv}, nil
	}})
	httpStatusCls.Dict.SetStr("__le__", &object.BuiltinFunc{Name: "__le__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.NotImplemented, nil
		}
		lv, rv, ok := httpStatusCmpVals(a)
		if !ok {
			return object.NotImplemented, nil
		}
		return &object.Bool{V: lv <= rv}, nil
	}})
	httpStatusCls.Dict.SetStr("__gt__", &object.BuiltinFunc{Name: "__gt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.NotImplemented, nil
		}
		lv, rv, ok := httpStatusCmpVals(a)
		if !ok {
			return object.NotImplemented, nil
		}
		return &object.Bool{V: lv > rv}, nil
	}})
	httpStatusCls.Dict.SetStr("__ge__", &object.BuiltinFunc{Name: "__ge__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.NotImplemented, nil
		}
		lv, rv, ok := httpStatusCmpVals(a)
		if !ok {
			return object.NotImplemented, nil
		}
		return &object.Bool{V: lv >= rv}, nil
	}})

	// Build all HTTPStatus members.
	type httpStatusDef struct {
		code   int
		name   string
		phrase string
		desc   string
	}
	statusDefs := []httpStatusDef{
		{100, "CONTINUE", "Continue", "Request received, please continue"},
		{101, "SWITCHING_PROTOCOLS", "Switching Protocols", "Switching to new protocol; obey Upgrade header"},
		{102, "PROCESSING", "Processing", "WebDAV; RFC 2518"},
		{103, "EARLY_HINTS", "Early Hints", "RFC 8297"},
		{200, "OK", "OK", "Request fulfilled, document follows"},
		{201, "CREATED", "Created", "Document created, URL follows"},
		{202, "ACCEPTED", "Accepted", "Request accepted, processing continues off-line"},
		{203, "NON_AUTHORITATIVE_INFORMATION", "Non-Authoritative Information", "Request fulfilled from cache"},
		{204, "NO_CONTENT", "No Content", "Request fulfilled, nothing follows"},
		{205, "RESET_CONTENT", "Reset Content", "Clear input form for further input"},
		{206, "PARTIAL_CONTENT", "Partial Content", "Partial content follows"},
		{207, "MULTI_STATUS", "Multi-Status", "WebDAV; RFC 4918"},
		{208, "ALREADY_REPORTED", "Already Reported", "WebDAV; RFC 5842"},
		{226, "IM_USED", "IM Used", "RFC 3229"},
		{300, "MULTIPLE_CHOICES", "Multiple Choices", "Object has several resources -- see URI list"},
		{301, "MOVED_PERMANENTLY", "Moved Permanently", "Object moved permanently -- see URI list"},
		{302, "FOUND", "Found", "Object moved temporarily -- see URI list"},
		{303, "SEE_OTHER", "See Other", "Object moved -- see Method and URL list"},
		{304, "NOT_MODIFIED", "Not Modified", "Document has not changed since given time"},
		{305, "USE_PROXY", "Use Proxy", "You must use proxy specified in Location to access this resource"},
		{307, "TEMPORARY_REDIRECT", "Temporary Redirect", "Object moved temporarily -- see URI list"},
		{308, "PERMANENT_REDIRECT", "Permanent Redirect", "Object moved permanently -- see URI list"},
		{400, "BAD_REQUEST", "Bad Request", "Bad request syntax or unsupported method"},
		{401, "UNAUTHORIZED", "Unauthorized", "No permission -- see authorization schemes"},
		{402, "PAYMENT_REQUIRED", "Payment Required", "No payment -- see charging schemes"},
		{403, "FORBIDDEN", "Forbidden", "Request forbidden -- authorization will not help"},
		{404, "NOT_FOUND", "Not Found", "Nothing matches the given URI"},
		{405, "METHOD_NOT_ALLOWED", "Method Not Allowed", "Specified method is invalid for this resource"},
		{406, "NOT_ACCEPTABLE", "Not Acceptable", "URI not available in preferred format"},
		{407, "PROXY_AUTHENTICATION_REQUIRED", "Proxy Authentication Required", "You must authenticate with this proxy before proceeding"},
		{408, "REQUEST_TIMEOUT", "Request Timeout", "Request timed out; try again later"},
		{409, "CONFLICT", "Conflict", "Request conflict"},
		{410, "GONE", "Gone", "URI no longer exists and has been permanently removed"},
		{411, "LENGTH_REQUIRED", "Length Required", "Client must specify Content-Length"},
		{412, "PRECONDITION_FAILED", "Precondition Failed", "Precondition in headers is false"},
		{413, "REQUEST_ENTITY_TOO_LARGE", "Request Entity Too Large", "Entity is too large"},
		{414, "REQUEST_URI_TOO_LONG", "Request-URI Too Long", "URI is too long"},
		{415, "UNSUPPORTED_MEDIA_TYPE", "Unsupported Media Type", "Entity body in unsupported format"},
		{416, "REQUESTED_RANGE_NOT_SATISFIABLE", "Requested Range Not Satisfiable", "Cannot satisfy request range"},
		{417, "EXPECTATION_FAILED", "Expectation Failed", "Expect condition could not be satisfied"},
		{418, "IM_A_TEAPOT", "I'm a Teapot", "Server refuses to brew coffee because it is a teapot"},
		{421, "MISDIRECTED_REQUEST", "Misdirected Request", "Server is not able to produce a response"},
		{422, "UNPROCESSABLE_ENTITY", "Unprocessable Entity", "WebDAV; RFC 4918"},
		{423, "LOCKED", "Locked", "WebDAV; RFC 4918"},
		{424, "FAILED_DEPENDENCY", "Failed Dependency", "WebDAV; RFC 4918"},
		{425, "TOO_EARLY", "Too Early", "RFC 8470"},
		{426, "UPGRADE_REQUIRED", "Upgrade Required", "Client should switch to a different protocol such as TLS/1.0"},
		{428, "PRECONDITION_REQUIRED", "Precondition Required", "The origin server requires the request to be conditional"},
		{429, "TOO_MANY_REQUESTS", "Too Many Requests", "The user has sent too many requests in a given amount of time ('rate limiting')"},
		{431, "REQUEST_HEADER_FIELDS_TOO_LARGE", "Request Header Fields Too Large", "The server is unwilling to process the request because its header fields are too large"},
		{451, "UNAVAILABLE_FOR_LEGAL_REASONS", "Unavailable For Legal Reasons", "The server is denying access to the resource as a consequence of a legal demand"},
		{500, "INTERNAL_SERVER_ERROR", "Internal Server Error", "Server got itself in trouble"},
		{501, "NOT_IMPLEMENTED", "Not Implemented", "Server does not support this operation"},
		{502, "BAD_GATEWAY", "Bad Gateway", "Invalid responses from another server/proxy"},
		{503, "SERVICE_UNAVAILABLE", "Service Unavailable", "The server cannot process the request due to a high load"},
		{504, "GATEWAY_TIMEOUT", "Gateway Timeout", "The gateway server did not receive a timely response"},
		{505, "HTTP_VERSION_NOT_SUPPORTED", "HTTP Version Not Supported", "Cannot fulfill request"},
		{506, "VARIANT_ALSO_NEGOTIATES", "Variant Also Negotiates", "RFC 2295"},
		{507, "INSUFFICIENT_STORAGE", "Insufficient Storage", "WebDAV; RFC 4918"},
		{508, "LOOP_DETECTED", "Loop Detected", "WebDAV; RFC 5842"},
		{510, "NOT_EXTENDED", "Not Extended", "RFC 2774"},
		{511, "NETWORK_AUTHENTICATION_REQUIRED", "Network Authentication Required", "RFC 6585"},
	}

	statusMembers := make([]*object.Instance, 0, len(statusDefs))
	statusMemberNames := make([]string, 0, len(statusDefs))
	statusMemberMap := object.NewDict()
	statusValMap := map[string]*object.Instance{}

	for _, def := range statusDefs {
		mem := &object.Instance{Class: httpStatusCls, Dict: object.NewDict()}
		iv := object.NewInt(int64(def.code))
		c := def.code
		mem.Dict.SetStr("_name_", &object.Str{V: def.name})
		mem.Dict.SetStr("name", &object.Str{V: def.name})
		mem.Dict.SetStr("_value_", iv)
		mem.Dict.SetStr("value", iv)
		mem.Dict.SetStr("phrase", &object.Str{V: def.phrase})
		mem.Dict.SetStr("description", &object.Str{V: def.desc})
		mem.Dict.SetStr("is_informational", &object.Bool{V: c >= 100 && c <= 199})
		mem.Dict.SetStr("is_success", &object.Bool{V: c >= 200 && c <= 299})
		mem.Dict.SetStr("is_redirection", &object.Bool{V: c >= 300 && c <= 399})
		mem.Dict.SetStr("is_client_error", &object.Bool{V: c >= 400 && c <= 499})
		mem.Dict.SetStr("is_server_error", &object.Bool{V: c >= 500 && c <= 599})
		mem.Dict.SetStr("is_integer", object.True)
		statusMembers = append(statusMembers, mem)
		statusMemberNames = append(statusMemberNames, def.name)
		statusMemberMap.SetStr(def.name, mem)
		statusValMap[object.EnumValueKey(iv)] = mem
		httpStatusCls.Dict.SetStr(def.name, mem)
	}

	httpStatusCls.EnumData = &object.EnumData{
		Members:     statusMembers,
		MemberMap:   statusMemberMap,
		ValMap:      statusValMap,
		MemberNames: statusMemberNames,
		BaseType:    "IntEnum",
	}
	httpStatusCls.Dict.SetStr("__members__", statusMemberMap)

	// HTTPStatus(200) → lookup by value
	httpStatusCls.Dict.SetStr("__new__", &object.BuiltinFunc{Name: "__new__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "HTTPStatus() requires a value argument")
		}
		val := a[1]
		key := object.EnumValueKey(val)
		if mem, ok := statusValMap[key]; ok {
			return mem, nil
		}
		n, _ := toInt64(val)
		return nil, object.Errorf(i.valueErr, "%d is not a valid HTTPStatus", n)
	}})

	// HTTPStatus['OK'] → subscript by name
	httpStatusCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "HTTPStatus[] requires a name")
		}
		nameStr, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "HTTPStatus[] key must be a string")
		}
		if mem, ok2 := statusMemberMap.GetStr(nameStr.V); ok2 {
			return mem, nil
		}
		return nil, object.Errorf(i.keyErr, "%s", nameStr.V)
	}})

	m.Dict.SetStr("HTTPStatus", httpStatusCls)

	// ── HTTPMethod ────────────────────────────────────────────────────────────

	httpMethodCls := &object.Class{Name: "HTTPMethod", Dict: object.NewDict()}

	methodMemberStrVal := func(inst *object.Instance) string {
		if v, ok := inst.Dict.GetStr("_value_"); ok {
			if s, ok := v.(*object.Str); ok {
				return s.V
			}
		}
		return ""
	}

	httpMethodCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if inst, ok := a[0].(*object.Instance); ok {
			return &object.Str{V: fmt.Sprintf("<HTTPMethod.%s: '%s'>", methodMemberStrVal(inst), methodMemberStrVal(inst))}, nil
		}
		return &object.Str{V: "<HTTPMethod>"}, nil
	}})
	httpMethodCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if inst, ok := a[0].(*object.Instance); ok {
			return &object.Str{V: methodMemberStrVal(inst)}, nil
		}
		return &object.Str{V: ""}, nil
	}})
	httpMethodCls.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		if a[0] == a[1] {
			return object.True, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.False, nil
		}
		lv := methodMemberStrVal(inst)
		switch rv := a[1].(type) {
		case *object.Str:
			return &object.Bool{V: lv == rv.V}, nil
		case *object.Instance:
			return &object.Bool{V: lv == methodMemberStrVal(rv)}, nil
		}
		return object.False, nil
	}})
	httpMethodCls.Dict.SetStr("__ne__", &object.BuiltinFunc{Name: "__ne__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.True, nil
		}
		if a[0] == a[1] {
			return object.False, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.True, nil
		}
		lv := methodMemberStrVal(inst)
		switch rv := a[1].(type) {
		case *object.Str:
			return &object.Bool{V: lv != rv.V}, nil
		case *object.Instance:
			return &object.Bool{V: lv != methodMemberStrVal(rv)}, nil
		}
		return object.True, nil
	}})
	httpMethodCls.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if inst, ok := a[0].(*object.Instance); ok {
			h := int64(0)
			for _, r := range methodMemberStrVal(inst) {
				h = h*31 + int64(r)
			}
			return object.NewInt(h), nil
		}
		return object.NewInt(0), nil
	}})

	// HTTPMethod('GET') → lookup by value
	httpMethodCls.Dict.SetStr("__new__", &object.BuiltinFunc{Name: "__new__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "HTTPMethod() requires a value argument")
		}
		valStr, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "HTTPMethod() value must be a string")
		}
		key := object.EnumValueKey(valStr)
		// methodValMap captured below
		_ = key
		return nil, object.Errorf(i.valueErr, "%s is not a valid HTTPMethod", valStr.V)
	}})

	type httpMethodDef struct {
		name string
		desc string
	}
	methodDefs := []httpMethodDef{
		{"CONNECT", "Establish a connection to the server."},
		{"DELETE", "Remove the target."},
		{"GET", "Retrieve the target."},
		{"HEAD", "Same as GET, but only retrieve the status line and header section."},
		{"OPTIONS", "Describe the communication options for the target."},
		{"PATCH", "Apply partial modifications to a target."},
		{"POST", "Perform target-specific processing with the request payload."},
		{"PUT", "Replace the target with the request payload."},
		{"TRACE", "Perform a message loop-back test along the path to the target."},
	}

	methodMembers := make([]*object.Instance, 0, len(methodDefs))
	methodMemberNames := make([]string, 0, len(methodDefs))
	methodMemberMap := object.NewDict()
	methodValMap := map[string]*object.Instance{}

	for _, def := range methodDefs {
		mem := &object.Instance{Class: httpMethodCls, Dict: object.NewDict()}
		sv := &object.Str{V: def.name}
		mem.Dict.SetStr("_name_", sv)
		mem.Dict.SetStr("name", sv)
		mem.Dict.SetStr("_value_", sv)
		mem.Dict.SetStr("value", sv)
		mem.Dict.SetStr("description", &object.Str{V: def.desc})
		methodMembers = append(methodMembers, mem)
		methodMemberNames = append(methodMemberNames, def.name)
		methodMemberMap.SetStr(def.name, mem)
		methodValMap[object.EnumValueKey(sv)] = mem
		httpMethodCls.Dict.SetStr(def.name, mem)
	}

	httpMethodCls.EnumData = &object.EnumData{
		Members:     methodMembers,
		MemberMap:   methodMemberMap,
		ValMap:      methodValMap,
		MemberNames: methodMemberNames,
		BaseType:    "StrEnum",
	}
	httpMethodCls.Dict.SetStr("__members__", methodMemberMap)

	// Fix __new__ now that methodValMap is populated.
	httpMethodCls.Dict.SetStr("__new__", &object.BuiltinFunc{Name: "__new__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "HTTPMethod() requires a value argument")
		}
		valStr, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "HTTPMethod() value must be a string")
		}
		key := object.EnumValueKey(valStr)
		if mem, ok2 := methodValMap[key]; ok2 {
			return mem, nil
		}
		return nil, object.Errorf(i.valueErr, "%s is not a valid HTTPMethod", valStr.V)
	}})

	// HTTPMethod['GET'] → subscript by name
	httpMethodCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "HTTPMethod[] requires a name")
		}
		nameStr, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "HTTPMethod[] key must be a string")
		}
		if mem, ok2 := methodMemberMap.GetStr(nameStr.V); ok2 {
			return mem, nil
		}
		return nil, object.Errorf(i.keyErr, "%s", nameStr.V)
	}})

	m.Dict.SetStr("HTTPMethod", httpMethodCls)

	return m
}

// httpStatusCmpVals extracts int64 values from two HTTPStatus members (or one
// member and one plain int) for ordering comparisons.
func httpStatusCmpVals(a []object.Object) (int64, int64, bool) {
	inst, ok := a[0].(*object.Instance)
	if !ok {
		return 0, 0, false
	}
	lv, lok := toInt64(a[0])
	if !lok {
		if v, ok2 := inst.Dict.GetStr("_value_"); ok2 {
			lv, lok = toInt64(v)
		}
	}
	if !lok {
		return 0, 0, false
	}
	rv, rok := toInt64(a[1])
	if !rok {
		if inst2, ok3 := a[1].(*object.Instance); ok3 {
			if v, ok4 := inst2.Dict.GetStr("_value_"); ok4 {
				rv, rok = toInt64(v)
			}
		}
	}
	return lv, rv, rok
}
