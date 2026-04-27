import unittest
import io


class TestBasicAssertions(unittest.TestCase):
    def test_basic(self):
        self.assertEqual(1, 1)
        self.assertNotEqual(1, 2)
        self.assertTrue(True)
        self.assertFalse(False)
        self.assertIsNone(None)
        self.assertIsNotNone(42)

    def test_identity(self):
        x = [1]
        self.assertIs(x, x)
        self.assertIsNot(x, [1])

    def test_membership(self):
        self.assertIn(1, [1, 2, 3])
        self.assertNotIn(4, [1, 2, 3])

    def test_isinstance_check(self):
        self.assertIsInstance(42, int)
        self.assertNotIsInstance(42, str)

    def test_comparisons(self):
        self.assertGreater(2, 1)
        self.assertGreaterEqual(2, 2)
        self.assertLess(1, 2)
        self.assertLessEqual(1, 1)

    def test_almost_equal(self):
        self.assertAlmostEqual(1.0, 1.000000001)
        self.assertNotAlmostEqual(1.0, 2.0)

    def test_raises(self):
        with self.assertRaises(ValueError):
            raise ValueError('expected')

    def test_fail_catch(self):
        try:
            self.fail('explicit fail')
        except AssertionError:
            pass

    def test_skip_catch(self):
        try:
            self.skipTest('skipping')
        except unittest.SkipTest:
            pass

    def test_sequence_dict(self):
        self.assertSequenceEqual([1, 2], [1, 2])
        self.assertListEqual([1, 2], [1, 2])
        self.assertTupleEqual((1, 2), (1, 2))
        self.assertDictEqual({'a': 1}, {'a': 1})

    def test_regex(self):
        self.assertRegex('hello world', r'hello')
        self.assertNotRegex('hello world', r'^world')

    def test_count_equal(self):
        self.assertCountEqual([1, 2, 3], [3, 1, 2])


def test_test_case_api():
    tc = TestBasicAssertions('test_basic')
    print(type(tc).__name__)
    print(tc.countTestCases())
    print(tc.id().endswith('TestBasicAssertions.test_basic'))
    print('test_test_case_api ok')


def test_suite():
    suite = unittest.TestSuite()
    suite.addTest(TestBasicAssertions('test_basic'))
    suite.addTest(TestBasicAssertions('test_identity'))
    print(type(suite).__name__)
    print(suite.countTestCases())
    print('test_suite ok')


def test_loader():
    loader = unittest.TestLoader()
    suite = loader.loadTestsFromTestCase(TestBasicAssertions)
    print(type(suite).__name__)
    print(suite.countTestCases() > 0)
    names = loader.getTestCaseNames(TestBasicAssertions)
    print(type(names).__name__)
    print(len(names) > 0)
    print('test_loader ok')


def test_result():
    result = unittest.TestResult()
    print(type(result).__name__)
    print(result.testsRun)
    print(result.wasSuccessful())
    print(len(result.errors))
    print(len(result.failures))
    print(len(result.skipped))
    print('test_result ok')


def test_run():
    result = unittest.TestResult()
    tc = TestBasicAssertions('test_basic')
    tc.run(result)
    print(result.wasSuccessful())
    print(result.testsRun)
    print('test_run ok')


def test_runner():
    suite = unittest.TestLoader().loadTestsFromTestCase(TestBasicAssertions)
    stream = io.StringIO()
    runner = unittest.TextTestRunner(stream=stream, verbosity=0)
    result = runner.run(suite)
    print(type(result).__name__)
    print(result.wasSuccessful())
    print('test_runner ok')


def test_skip_decorators():
    @unittest.skip('always skip')
    def skipped():
        pass

    @unittest.skipIf(True, 'condition true')
    def skipped_if():
        pass

    @unittest.skipUnless(False, 'condition false')
    def skipped_unless():
        pass

    print(skipped.__unittest_skip__)
    print(skipped.__unittest_skip_why__)
    print(skipped_if.__unittest_skip__)
    print(skipped_unless.__unittest_skip__)
    print('test_skip_decorators ok')


def test_expected_failure():
    @unittest.expectedFailure
    def might_fail():
        pass
    print(hasattr(might_fail, '__unittest_expecting_failure__'))
    print(might_fail.__unittest_expecting_failure__)
    print('test_expected_failure ok')


def test_main_api():
    print(hasattr(unittest, 'main'))
    print(hasattr(unittest, 'defaultTestLoader'))
    print(type(unittest.defaultTestLoader).__name__)
    print('test_main_api ok')


test_test_case_api()
test_suite()
test_loader()
test_result()
test_run()
test_runner()
test_skip_decorators()
test_expected_failure()
test_main_api()
