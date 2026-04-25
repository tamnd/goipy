import webbrowser


def test_error_class():
    assert issubclass(webbrowser.Error, Exception)
    try:
        raise webbrowser.Error("test error")
    except webbrowser.Error as e:
        assert str(e) == "test error"
    print("error_class ok")


def test_controller_class():
    c = webbrowser.Controller("mybrowser")
    assert c.name == "mybrowser"
    assert isinstance(c, webbrowser.Controller)
    assert hasattr(c, "open")
    assert hasattr(c, "open_new")
    assert hasattr(c, "open_new_tab")
    print("controller_class ok")


def test_register_get():
    class MockBrowser:
        name = "mock"

        def open(self, url, new=0, autoraise=True):
            return True

        def open_new(self, url):
            return self.open(url, 1)

        def open_new_tab(self, url):
            return self.open(url, 2)

    mock = MockBrowser()
    webbrowser.register("mock", None, mock)
    b = webbrowser.get("mock")
    assert b is mock
    assert b.open("http://example.com") == True
    assert b.open_new("http://example.com") == True
    assert b.open_new_tab("http://example.com") == True
    print("register_get ok")


def test_get_error():
    try:
        webbrowser.get("nonexistent_browser_xyz_123")
        assert False, "should have raised"
    except webbrowser.Error:
        pass
    print("get_error ok")


def test_module_functions():
    assert callable(webbrowser.open)
    assert callable(webbrowser.open_new)
    assert callable(webbrowser.open_new_tab)
    assert callable(webbrowser.get)
    assert callable(webbrowser.register)
    print("module_functions ok")


def test_open_returns_bool():
    result = webbrowser.open("http://example.com")
    assert isinstance(result, bool)
    print("open_returns_bool ok")


def test_open_new_returns_bool():
    result = webbrowser.open_new("http://example.com")
    assert isinstance(result, bool)
    print("open_new_returns_bool ok")


def test_open_new_tab_returns_bool():
    result = webbrowser.open_new_tab("http://example.com")
    assert isinstance(result, bool)
    print("open_new_tab_returns_bool ok")


def test_get_default():
    b = webbrowser.get()
    assert b is not None
    assert hasattr(b, "open")
    print("get_default ok")


def test_preferred_browser():
    class PreferredBrowser:
        name = "preferred_test"

        def open(self, url, new=0, autoraise=True):
            return True

        def open_new(self, url):
            return True

        def open_new_tab(self, url):
            return True

    pb = PreferredBrowser()
    webbrowser.register("preferred_test", None, pb, preferred=True)
    default_b = webbrowser.get()
    assert default_b is pb
    print("preferred_browser ok")


def test_constructor_register():
    class FactoryBrowser:
        name = "factory"

        def open(self, url, new=0, autoraise=True):
            return True

        def open_new(self, url):
            return True

        def open_new_tab(self, url):
            return True

    webbrowser.register("factory", FactoryBrowser)
    b = webbrowser.get("factory")
    assert isinstance(b, FactoryBrowser)
    print("constructor_register ok")


test_error_class()
test_controller_class()
test_register_get()
test_get_error()
test_module_functions()
test_open_returns_bool()
test_open_new_returns_bool()
test_open_new_tab_returns_bool()
test_get_default()
test_preferred_browser()
test_constructor_register()
