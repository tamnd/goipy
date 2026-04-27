import timeit


def run(name, fn):
    try:
        fn()
        print(f"{name}: OK")
    except Exception as e:
        print(f"{name}: FAIL ({e})")


# basic import
def test_import():
    import timeit as t
    assert t is not None

run("test_import", test_import)


# default_timer is callable and returns a float
def test_default_timer():
    t = timeit.default_timer()
    assert isinstance(t, float), f"expected float, got {type(t)}"
    assert t >= 0.0

run("test_default_timer", test_default_timer)


# Timer constructor with callable
def test_timer_constructor():
    timer = timeit.Timer(lambda: None)
    assert timer is not None

run("test_timer_constructor", test_timer_constructor)


# Timer constructor with string stmt (no crash)
def test_timer_string_stmt():
    timer = timeit.Timer("pass")
    assert timer is not None

run("test_timer_string_stmt", test_timer_string_stmt)


# Timer.timeit returns a non-negative float
def test_timer_timeit_returns_float():
    timer = timeit.Timer(lambda: None)
    result = timer.timeit(number=10)
    assert isinstance(result, float), f"expected float, got {type(result)}"
    assert result >= 0.0

run("test_timer_timeit_returns_float", test_timer_timeit_returns_float)


# Timer.timeit with default number works
def test_timer_timeit_default_number():
    timer = timeit.Timer(lambda: None)
    result = timer.timeit()
    assert isinstance(result, float)
    assert result >= 0.0

run("test_timer_timeit_default_number", test_timer_timeit_default_number)


# Timer with callable setup
def test_timer_callable_setup():
    calls = []
    timer = timeit.Timer(lambda: calls.append(1), setup=lambda: calls.append(0))
    timer.timeit(number=3)
    # setup called once, stmt called 3 times
    assert calls[0] == 0
    assert len([x for x in calls if x == 1]) == 3

run("test_timer_callable_setup", test_timer_callable_setup)


# Timer.repeat returns list of floats
def test_timer_repeat():
    timer = timeit.Timer(lambda: None)
    result = timer.repeat(repeat=3, number=5)
    assert isinstance(result, list), f"expected list, got {type(result)}"
    assert len(result) == 3

run("test_timer_repeat", test_timer_repeat)


# Each repeat element is a float >= 0
def test_timer_repeat_elements():
    timer = timeit.Timer(lambda: None)
    result = timer.repeat(repeat=2, number=5)
    for x in result:
        assert isinstance(x, float), f"expected float, got {type(x)}"
        assert x >= 0.0

run("test_timer_repeat_elements", test_timer_repeat_elements)


# Timer.repeat default repeat=5
def test_timer_repeat_default():
    timer = timeit.Timer(lambda: None)
    result = timer.repeat(number=2)
    assert isinstance(result, list)
    assert len(result) == 5

run("test_timer_repeat_default", test_timer_repeat_default)


# Timer.autorange returns a 2-tuple (number, time_taken)
def test_timer_autorange():
    timer = timeit.Timer(lambda: None)
    result = timer.autorange()
    assert isinstance(result, tuple), f"expected tuple, got {type(result)}"
    assert len(result) == 2

run("test_timer_autorange", test_timer_autorange)


# autorange number is a positive int
def test_timer_autorange_number_positive():
    timer = timeit.Timer(lambda: None)
    number, time_taken = timer.autorange()
    assert isinstance(number, int), f"expected int, got {type(number)}"
    assert number > 0

run("test_timer_autorange_number_positive", test_timer_autorange_number_positive)


# autorange time_taken is a non-negative float
def test_timer_autorange_time_taken():
    timer = timeit.Timer(lambda: None)
    number, time_taken = timer.autorange()
    assert isinstance(time_taken, float), f"expected float, got {type(time_taken)}"
    assert time_taken >= 0.0

run("test_timer_autorange_time_taken", test_timer_autorange_time_taken)


# autorange callback is invoked
def test_timer_autorange_callback():
    called = []
    def cb(number, time_taken):
        called.append((number, time_taken))
    timer = timeit.Timer(lambda: None)
    timer.autorange(callback=cb)
    assert len(called) >= 1
    for number, time_taken in called:
        assert isinstance(number, int)
        assert isinstance(time_taken, float)

run("test_timer_autorange_callback", test_timer_autorange_callback)


# Timer.print_exc does not crash
def test_timer_print_exc():
    timer = timeit.Timer(lambda: None)
    timer.print_exc()  # no-op stub

run("test_timer_print_exc", test_timer_print_exc)


# Module-level timeit() returns float
def test_module_timeit():
    result = timeit.timeit(lambda: None, number=10)
    assert isinstance(result, float)
    assert result >= 0.0

run("test_module_timeit", test_module_timeit)


# Module-level timeit with string stmt
def test_module_timeit_string():
    result = timeit.timeit("pass", number=10)
    assert isinstance(result, float)
    assert result >= 0.0

run("test_module_timeit_string", test_module_timeit_string)


# Module-level repeat() returns list
def test_module_repeat():
    result = timeit.repeat(lambda: None, repeat=3, number=5)
    assert isinstance(result, list)
    assert len(result) == 3

run("test_module_repeat", test_module_repeat)


# Module-level repeat with string
def test_module_repeat_string():
    result = timeit.repeat("pass", repeat=2, number=5)
    assert isinstance(result, list)
    assert len(result) == 2

run("test_module_repeat_string", test_module_repeat_string)


# Timer callable counts correctly
def test_timer_call_count():
    counter = [0]
    def inc():
        counter[0] += 1
    timer = timeit.Timer(inc)
    timer.timeit(number=7)
    assert counter[0] == 7, f"expected 7, got {counter[0]}"

run("test_timer_call_count", test_timer_call_count)


# Timer with no arguments uses default stmt
def test_timer_default_stmt():
    timer = timeit.Timer()
    result = timer.timeit(number=5)
    assert isinstance(result, float)

run("test_timer_default_stmt", test_timer_default_stmt)


# Timer.timeit with number=0 returns 0.0 or very small float
def test_timer_timeit_zero():
    timer = timeit.Timer(lambda: None)
    result = timer.timeit(number=0)
    assert isinstance(result, float)
    assert result >= 0.0

run("test_timer_timeit_zero", test_timer_timeit_zero)


# timeit.Timer is accessible as a class
def test_timer_class():
    assert timeit.Timer is not None
    t = timeit.Timer(lambda: None)
    assert isinstance(t, timeit.Timer)

run("test_timer_class", test_timer_class)
