import doctest


def test_constants():
    print(doctest.ELLIPSIS)
    print(doctest.NORMALIZE_WHITESPACE)
    print(doctest.IGNORE_EXCEPTION_DETAIL)
    print(doctest.DONT_ACCEPT_BLANKLINE)
    print(doctest.DONT_ACCEPT_TRUE_FOR_1)
    print(doctest.SKIP)
    print(doctest.FAIL_FAST)
    print(doctest.REPORT_UDIFF)
    print(doctest.REPORT_CDIFF)
    print(doctest.REPORT_NDIFF)
    print(doctest.REPORT_ONLY_FIRST_FAILURE)
    print(doctest.BLANKLINE_MARKER)
    print(doctest.ELLIPSIS_MARKER)
    print('test_constants ok')


def test_register_optionflag():
    flag1 = doctest.register_optionflag('MY_FLAG_A')
    flag2 = doctest.register_optionflag('MY_FLAG_B')
    print(type(flag1).__name__)
    print(flag1 > 0)
    print(flag2 > flag1)
    flag1b = doctest.register_optionflag('MY_FLAG_A')
    print(flag1 == flag1b)
    print('test_register_optionflag ok')


def test_example():
    ex = doctest.Example('1+1\n', '2\n', lineno=3)
    print(type(ex).__name__)
    print(repr(ex.source))
    print(repr(ex.want))
    print(ex.lineno)
    print(ex.indent)
    print(type(ex.options).__name__)
    print('test_example ok')


def test_doctest_class():
    ex = doctest.Example('1+1\n', '2\n')
    dt = doctest.DocTest([ex], {}, 'mytest', 'file.py', 1, 'docstring')
    print(type(dt).__name__)
    print(dt.name)
    print(dt.filename)
    print(dt.lineno)
    print(dt.docstring)
    print(len(dt.examples))
    print(type(dt.globs).__name__)
    print('DocTest' in repr(dt))
    print('mytest' in repr(dt))
    print('test_doctest_class ok')


def test_parser_get_examples():
    parser = doctest.DocTestParser()
    docstring = '\n    >>> 1 + 1\n    2\n    >>> print("hello")\n    hello\n    '
    examples = parser.get_examples(docstring)
    print(type(examples).__name__)
    print(len(examples))
    ex0 = examples[0]
    print(type(ex0).__name__)
    print(repr(ex0.source))
    print(repr(ex0.want))
    print('test_parser_get_examples ok')


def test_parser_parse():
    parser = doctest.DocTestParser()
    docstring = 'Before\n>>> 1+1\n2\nAfter'
    items = parser.parse(docstring)
    print(type(items).__name__)
    has_str = any(isinstance(x, str) for x in items)
    has_example = any(isinstance(x, doctest.Example) for x in items)
    print(has_str)
    print(has_example)
    print('test_parser_parse ok')


def test_parser_get_doctest():
    parser = doctest.DocTestParser()
    docstring = '>>> 1+1\n2\n'
    dt = parser.get_doctest(docstring, {}, 'mymod', 'mymod.py', 0)
    print(type(dt).__name__)
    print(dt.name)
    print(len(dt.examples))
    print('test_parser_get_doctest ok')


def test_output_checker():
    checker = doctest.OutputChecker()
    print(type(checker).__name__)
    print(checker.check_output('2\n', '2\n', 0))
    print(checker.check_output('2\n', '3\n', 0))
    print(checker.check_output('foo...\n', 'foobar\n', doctest.ELLIPSIS))
    print(checker.check_output('a b\n', 'a  b\n', doctest.NORMALIZE_WHITESPACE))
    print('test_output_checker ok')


def test_test_results():
    r = doctest.TestResults(failed=0, attempted=3)
    print(type(r).__name__)
    print(r.failed)
    print(r.attempted)
    print('test_test_results ok')


def test_runner():
    parser = doctest.DocTestParser()
    docstring = '>>> 1+1\n2\n>>> print("hi")\nhi\n'
    dt = parser.get_doctest(docstring, {}, 'mytest', '<string>', 0)
    runner = doctest.DocTestRunner(verbose=False)
    results = runner.run(dt)
    print(type(results).__name__)
    print(results.failed)
    print(results.attempted)
    print('test_runner ok')


def test_finder():
    def myfunc():
        '''
        >>> 1+1
        2
        '''
        pass
    finder = doctest.DocTestFinder()
    tests = finder.find(myfunc, name='myfunc')
    print(type(tests).__name__)
    print(len(tests) >= 1)
    dt = tests[0]
    print(type(dt).__name__)
    print(dt.name)
    print(len(dt.examples))
    print('test_finder ok')


def test_exceptions():
    ex = doctest.Example('1+1\n', '2\n')
    dt = doctest.DocTest([], {}, 'test', 'file', 0, '')
    fail = doctest.DocTestFailure(dt, ex, 'got')
    print(type(fail).__name__)
    print(fail.test is dt)
    print(fail.example is ex)
    print(fail.got)
    unexp = doctest.UnexpectedException(dt, ex, (None, None, None))
    print(type(unexp).__name__)
    print(unexp.test is dt)
    print('test_exceptions ok')


def test_script_from_examples():
    docstring = '\n    >>> x = 1\n    >>> print(x)\n    1\n    '
    script = doctest.script_from_examples(docstring)
    print(type(script).__name__)
    print('x = 1' in script)
    print('print(x)' in script)
    print('test_script_from_examples ok')


def test_testmod():
    results = doctest.testmod(verbose=False)
    print(type(results).__name__)
    print(results.failed == 0)
    print('test_testmod ok')


test_constants()
test_register_optionflag()
test_example()
test_doctest_class()
test_parser_get_examples()
test_parser_parse()
test_parser_get_doctest()
test_output_checker()
test_test_results()
test_runner()
test_finder()
test_exceptions()
test_script_from_examples()
test_testmod()
