import http


def test_http_status_value():
    assert http.HTTPStatus.OK.value == 200
    assert http.HTTPStatus.NOT_FOUND.value == 404
    assert http.HTTPStatus.INTERNAL_SERVER_ERROR.value == 500
    assert http.HTTPStatus.CONTINUE.value == 100
    print("http_status_value ok")


def test_http_status_phrase():
    assert http.HTTPStatus.OK.phrase == "OK"
    assert http.HTTPStatus.NOT_FOUND.phrase == "Not Found"
    assert http.HTTPStatus.INTERNAL_SERVER_ERROR.phrase == "Internal Server Error"
    assert http.HTTPStatus.CONTINUE.phrase == "Continue"
    print("http_status_phrase ok")


def test_http_status_description():
    assert "fulfilled" in http.HTTPStatus.OK.description
    assert "matches" in http.HTTPStatus.NOT_FOUND.description
    assert "trouble" in http.HTTPStatus.INTERNAL_SERVER_ERROR.description
    print("http_status_description ok")


def test_http_status_name():
    assert http.HTTPStatus.OK.name == "OK"
    assert http.HTTPStatus.NOT_FOUND.name == "NOT_FOUND"
    assert http.HTTPStatus.CREATED.name == "CREATED"
    print("http_status_name ok")


def test_http_status_is_informational():
    assert http.HTTPStatus.CONTINUE.is_informational == True
    assert http.HTTPStatus.SWITCHING_PROTOCOLS.is_informational == True
    assert http.HTTPStatus.OK.is_informational == False
    assert http.HTTPStatus.NOT_FOUND.is_informational == False
    print("http_status_is_informational ok")


def test_http_status_is_success():
    assert http.HTTPStatus.OK.is_success == True
    assert http.HTTPStatus.CREATED.is_success == True
    assert http.HTTPStatus.NO_CONTENT.is_success == True
    assert http.HTTPStatus.NOT_FOUND.is_success == False
    assert http.HTTPStatus.CONTINUE.is_success == False
    print("http_status_is_success ok")


def test_http_status_is_redirection():
    assert http.HTTPStatus.MOVED_PERMANENTLY.is_redirection == True
    assert http.HTTPStatus.FOUND.is_redirection == True
    assert http.HTTPStatus.NOT_MODIFIED.is_redirection == True
    assert http.HTTPStatus.OK.is_redirection == False
    print("http_status_is_redirection ok")


def test_http_status_is_client_error():
    assert http.HTTPStatus.BAD_REQUEST.is_client_error == True
    assert http.HTTPStatus.NOT_FOUND.is_client_error == True
    assert http.HTTPStatus.UNAUTHORIZED.is_client_error == True
    assert http.HTTPStatus.OK.is_client_error == False
    assert http.HTTPStatus.INTERNAL_SERVER_ERROR.is_client_error == False
    print("http_status_is_client_error ok")


def test_http_status_is_server_error():
    assert http.HTTPStatus.INTERNAL_SERVER_ERROR.is_server_error == True
    assert http.HTTPStatus.SERVICE_UNAVAILABLE.is_server_error == True
    assert http.HTTPStatus.BAD_GATEWAY.is_server_error == True
    assert http.HTTPStatus.OK.is_server_error == False
    assert http.HTTPStatus.NOT_FOUND.is_server_error == False
    print("http_status_is_server_error ok")


def test_http_status_by_value():
    s = http.HTTPStatus(200)
    assert s is http.HTTPStatus.OK
    s404 = http.HTTPStatus(404)
    assert s404 is http.HTTPStatus.NOT_FOUND
    print("http_status_by_value ok")


def test_http_status_by_name():
    s = http.HTTPStatus["OK"]
    assert s is http.HTTPStatus.OK
    s404 = http.HTTPStatus["NOT_FOUND"]
    assert s404 is http.HTTPStatus.NOT_FOUND
    print("http_status_by_name ok")


def test_http_status_eq_int():
    assert http.HTTPStatus.OK == 200
    assert http.HTTPStatus.NOT_FOUND == 404
    assert http.HTTPStatus.OK != 404
    assert 200 == http.HTTPStatus.OK
    print("http_status_eq_int ok")


def test_http_status_repr():
    r = repr(http.HTTPStatus.OK)
    assert r == "<HTTPStatus.OK: 200>", repr(r)
    r2 = repr(http.HTTPStatus.NOT_FOUND)
    assert r2 == "<HTTPStatus.NOT_FOUND: 404>", repr(r2)
    print("http_status_repr ok")


def test_http_status_str():
    assert str(http.HTTPStatus.OK) == "200", repr(str(http.HTTPStatus.OK))
    assert str(http.HTTPStatus.NOT_FOUND) == "404"
    print("http_status_str ok")


def test_http_status_int():
    assert int(http.HTTPStatus.OK) == 200
    assert int(http.HTTPStatus.NOT_FOUND) == 404
    print("http_status_int ok")


def test_http_status_ordering():
    assert http.HTTPStatus.OK < http.HTTPStatus.NOT_FOUND
    assert http.HTTPStatus.NOT_FOUND > http.HTTPStatus.OK
    assert http.HTTPStatus.OK <= http.HTTPStatus.OK
    assert http.HTTPStatus.OK <= http.HTTPStatus.NOT_FOUND
    assert http.HTTPStatus.NOT_FOUND >= http.HTTPStatus.NOT_FOUND
    print("http_status_ordering ok")


def test_http_status_ordering_with_int():
    assert http.HTTPStatus.OK < 404
    assert http.HTTPStatus.NOT_FOUND > 200
    assert http.HTTPStatus.OK >= 200
    print("http_status_ordering_with_int ok")


def test_http_status_iteration():
    members = list(http.HTTPStatus)
    assert len(members) > 0
    assert http.HTTPStatus.CONTINUE in members
    assert http.HTTPStatus.OK in members
    assert http.HTTPStatus.NOT_FOUND in members
    assert http.HTTPStatus.INTERNAL_SERVER_ERROR in members
    # First member is CONTINUE (100), last is NETWORK_AUTHENTICATION_REQUIRED (511)
    assert members[0] is http.HTTPStatus.CONTINUE
    assert members[-1] is http.HTTPStatus.NETWORK_AUTHENTICATION_REQUIRED
    print("http_status_iteration ok")


def test_http_status_members_dict():
    members = http.HTTPStatus.__members__
    assert "OK" in members
    assert "NOT_FOUND" in members
    assert members["OK"] is http.HTTPStatus.OK
    print("http_status_members_dict ok")


def test_http_status_all_1xx():
    codes_1xx = [s for s in http.HTTPStatus if s.is_informational]
    assert len(codes_1xx) >= 3
    for s in codes_1xx:
        assert 100 <= s.value <= 199
    print("http_status_all_1xx ok")


def test_http_status_all_5xx():
    codes_5xx = [s for s in http.HTTPStatus if s.is_server_error]
    assert len(codes_5xx) >= 5
    for s in codes_5xx:
        assert 500 <= s.value <= 599
    print("http_status_all_5xx ok")


def test_http_status_teapot():
    t = http.HTTPStatus.IM_A_TEAPOT
    assert t.value == 418
    assert "Teapot" in t.phrase
    print("http_status_teapot ok")


def test_http_method_value():
    assert http.HTTPMethod.GET.value == "GET"
    assert http.HTTPMethod.POST.value == "POST"
    assert http.HTTPMethod.DELETE.value == "DELETE"
    print("http_method_value ok")


def test_http_method_name():
    assert http.HTTPMethod.GET.name == "GET"
    assert http.HTTPMethod.POST.name == "POST"
    print("http_method_name ok")


def test_http_method_description():
    assert "Retrieve" in http.HTTPMethod.GET.description
    assert "target-specific" in http.HTTPMethod.POST.description
    assert "connection" in http.HTTPMethod.CONNECT.description
    print("http_method_description ok")


def test_http_method_str():
    assert str(http.HTTPMethod.GET) == "GET"
    assert str(http.HTTPMethod.POST) == "POST"
    print("http_method_str ok")


def test_http_method_eq_str():
    assert http.HTTPMethod.GET == "GET"
    assert http.HTTPMethod.POST == "POST"
    assert http.HTTPMethod.DELETE != "GET"
    print("http_method_eq_str ok")


def test_http_method_by_value():
    m = http.HTTPMethod("GET")
    assert m is http.HTTPMethod.GET
    m2 = http.HTTPMethod("POST")
    assert m2 is http.HTTPMethod.POST
    print("http_method_by_value ok")


def test_http_method_by_name():
    m = http.HTTPMethod["GET"]
    assert m is http.HTTPMethod.GET
    print("http_method_by_name ok")


def test_http_method_iteration():
    methods = list(http.HTTPMethod)
    names = [m.value for m in methods]
    assert "GET" in names
    assert "POST" in names
    assert "PUT" in names
    assert "DELETE" in names
    assert "PATCH" in names
    assert "HEAD" in names
    assert "OPTIONS" in names
    assert "CONNECT" in names
    assert "TRACE" in names
    assert "QUERY" not in names
    print("http_method_iteration ok")


def test_http_method_members_dict():
    members = http.HTTPMethod.__members__
    assert "GET" in members
    assert "POST" in members
    assert members["GET"] is http.HTTPMethod.GET
    print("http_method_members_dict ok")


def test_exports():
    assert http.HTTPStatus is not None
    assert http.HTTPMethod is not None
    assert isinstance(http.HTTPStatus.OK, http.HTTPStatus)
    assert isinstance(http.HTTPMethod.GET, http.HTTPMethod)
    print("exports ok")


test_http_status_value()
test_http_status_phrase()
test_http_status_description()
test_http_status_name()
test_http_status_is_informational()
test_http_status_is_success()
test_http_status_is_redirection()
test_http_status_is_client_error()
test_http_status_is_server_error()
test_http_status_by_value()
test_http_status_by_name()
test_http_status_eq_int()
test_http_status_repr()
test_http_status_str()
test_http_status_int()
test_http_status_ordering()
test_http_status_ordering_with_int()
test_http_status_iteration()
test_http_status_members_dict()
test_http_status_all_1xx()
test_http_status_all_5xx()
test_http_status_teapot()
test_http_method_value()
test_http_method_name()
test_http_method_description()
test_http_method_str()
test_http_method_eq_str()
test_http_method_by_value()
test_http_method_by_name()
test_http_method_iteration()
test_http_method_members_dict()
test_exports()
