import gettext
import io
import struct


def make_mo(translations, plural_forms='nplurals=2; plural=(n != 1);'):
    """Build a minimal .mo binary from a {orig: trans} dict."""
    header_trans = (
        'Content-Type: text/plain; charset=UTF-8\n'
        'Plural-Forms: ' + plural_forms + '\n'
    )
    entries = [('', header_trans)] + sorted(
        (k, v) for k, v in translations.items()
    )
    n = len(entries)
    orig_table_off = 28
    trans_table_off = orig_table_off + n * 8
    strings_start = trans_table_off + n * 8

    orig_bytes = [k.encode('utf-8') for k, v in entries]
    trans_bytes = [v.encode('utf-8') for k, v in entries]

    pos = strings_start
    orig_off = []
    for b in orig_bytes:
        orig_off.append((len(b), pos))
        pos += len(b) + 1

    trans_off = []
    for b in trans_bytes:
        trans_off.append((len(b), pos))
        pos += len(b) + 1

    data = struct.pack('<IIIIIII', 0x950412de, 0, n,
                       orig_table_off, trans_table_off, 0, 28)
    for length, offset in orig_off:
        data += struct.pack('<II', length, offset)
    for length, offset in trans_off:
        data += struct.pack('<II', length, offset)
    for b in orig_bytes:
        data += b + b'\x00'
    for b in trans_bytes:
        data += b + b'\x00'
    return data


def test_null_translations():
    t = gettext.NullTranslations()
    print(t.gettext('hello'))
    print(t.gettext('world'))
    print(t.ngettext('apple', 'apples', 1))
    print(t.ngettext('apple', 'apples', 2))
    print(t.ngettext('apple', 'apples', 0))
    print(t.pgettext('fruit', 'apple'))
    print(t.npgettext('fruit', 'apple', 'apples', 1))
    print(t.npgettext('fruit', 'apple', 'apples', 5))
    print(t.charset() is None)
    print(t.info())
    print('test_null_translations ok')


def test_gnu_translations():
    mo = make_mo({'hello': 'hola', 'world': 'mundo', 'good morning': 'buenos dias'})
    t = gettext.GNUTranslations(io.BytesIO(mo))
    print(t.gettext('hello'))
    print(t.gettext('world'))
    print(t.gettext('good morning'))
    print(t.gettext('unknown'))
    print(t.charset())
    print('test_gnu_translations ok')


def test_gnu_ngettext():
    mo = make_mo({
        'apple\x00apples': 'manzana\x00manzanas',
        'item\x00items': 'elemento\x00elementos',
    })
    t = gettext.GNUTranslations(io.BytesIO(mo))
    print(t.ngettext('apple', 'apples', 1))
    print(t.ngettext('apple', 'apples', 2))
    print(t.ngettext('apple', 'apples', 0))
    print(t.ngettext('item', 'items', 1))
    print(t.ngettext('item', 'items', 5))
    print(t.ngettext('x', 'xs', 2))
    print('test_gnu_ngettext ok')


def test_gnu_pgettext():
    mo = make_mo({
        'fruit\x04apple': 'manzana',
        'tool\x04iron': 'hierro',
    })
    t = gettext.GNUTranslations(io.BytesIO(mo))
    print(t.pgettext('fruit', 'apple'))
    print(t.pgettext('tool', 'iron'))
    print(t.pgettext('food', 'apple'))
    print('test_gnu_pgettext ok')


def test_fallback():
    mo = make_mo({'hello': 'hola'})
    t1 = gettext.GNUTranslations(io.BytesIO(mo))
    t2 = gettext.NullTranslations()
    t1.add_fallback(t2)
    print(t1.gettext('hello'))
    print(t1.gettext('unknown'))
    print('test_fallback ok')


def test_install():
    t = gettext.NullTranslations()
    t.install()
    print(_('hello'))
    print(_('world'))
    print('test_install ok')


def test_module_level():
    print(gettext.gettext('hello'))
    print(gettext.ngettext('item', 'items', 1))
    print(gettext.ngettext('item', 'items', 5))
    print(gettext.pgettext('ctx', 'hello'))
    print(gettext.npgettext('ctx', 'item', 'items', 2))
    print('test_module_level ok')


def test_textdomain():
    old = gettext.textdomain()
    print(old)
    gettext.textdomain('myapp')
    print(gettext.textdomain())
    gettext.textdomain(old)
    print(gettext.textdomain())
    gettext.bindtextdomain('myapp', '/tmp/locale')
    print('test_textdomain ok')


test_null_translations()
test_gnu_translations()
test_gnu_ngettext()
test_gnu_pgettext()
test_fallback()
test_install()
test_module_level()
test_textdomain()
