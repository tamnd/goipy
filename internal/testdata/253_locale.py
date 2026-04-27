import locale


def test_constants():
    print(locale.LC_ALL)
    print(locale.LC_CTYPE)
    print(locale.LC_COLLATE)
    print(locale.LC_MONETARY)
    print(locale.LC_NUMERIC)
    print(locale.LC_TIME)
    print(locale.CHAR_MAX)
    print('test_constants ok')


def test_error():
    try:
        raise locale.Error('bad locale')
    except locale.Error as e:
        print(type(e).__name__)
        print(str(e))
    try:
        locale.setlocale(locale.LC_ALL, 'invalid_xyz_locale_999')
    except locale.Error:
        print('setlocale error caught')
    print('test_error ok')


def test_setlocale():
    locale.setlocale(locale.LC_ALL, 'C')
    r = locale.setlocale(locale.LC_ALL)
    print(type(r).__name__)
    print(r)
    r2 = locale.setlocale(locale.LC_ALL, 'C')
    print(r2)
    print('test_setlocale ok')


def test_localeconv():
    locale.setlocale(locale.LC_ALL, 'C')
    lc = locale.localeconv()
    print(lc['decimal_point'])
    print(repr(lc['thousands_sep']))
    print(lc['grouping'])
    print(lc['frac_digits'])
    print(lc['int_frac_digits'])
    print('test_localeconv ok')


def test_atof_atoi():
    locale.setlocale(locale.LC_ALL, 'C')
    print(locale.atof('3.14'))
    print(locale.atof('  2.718  '))
    print(locale.atoi('42'))
    print(locale.atoi('  -7  '))
    print('test_atof_atoi ok')


def test_format_string():
    locale.setlocale(locale.LC_ALL, 'C')
    print(locale.format_string('%d', 1000))
    print(locale.format_string('%.2f', 3.14159))
    print(locale.format_string('%s %s', ('hello', 'world')))
    print('test_format_string ok')


def test_strcoll():
    locale.setlocale(locale.LC_ALL, 'C')
    print(locale.strcoll('abc', 'abc'))
    print(locale.strcoll('abc', 'abd') < 0)
    print(locale.strcoll('abd', 'abc') > 0)
    print(locale.strxfrm('hello'))
    print('test_strcoll ok')


def test_delocalize():
    locale.setlocale(locale.LC_ALL, 'C')
    print(locale.delocalize('3.14'))
    print(locale.delocalize('1000'))
    print('test_delocalize ok')


def test_normalize():
    print(locale.normalize('C'))
    print(locale.normalize('en_US'))
    print(locale.normalize('en_US.UTF-8'))
    print('test_normalize ok')


def test_getencoding():
    enc = locale.getencoding()
    print(type(enc).__name__)
    print(isinstance(enc, str))
    print('test_getencoding ok')


def test_getpreferredencoding():
    enc = locale.getpreferredencoding(False)
    print(type(enc).__name__)
    print(isinstance(enc, str))
    print('test_getpreferredencoding ok')


test_constants()
test_error()
test_setlocale()
test_localeconv()
test_atof_atoi()
test_format_string()
test_strcoll()
test_delocalize()
test_normalize()
test_getencoding()
test_getpreferredencoding()
