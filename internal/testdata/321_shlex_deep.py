import shlex

# ── split: basic whitespace ────────────────────────────────────────────────────
print(shlex.split('') == [])
print(shlex.split('foo') == ['foo'])
print(shlex.split('foo bar baz') == ['foo', 'bar', 'baz'])
print(shlex.split('  foo  bar  ') == ['foo', 'bar'])
print(shlex.split('foo\tbar') == ['foo', 'bar'])

# ── split: single quotes ───────────────────────────────────────────────────────
print(shlex.split("'foo bar'") == ['foo bar'])
print(shlex.split("foo 'bar baz'") == ['foo', 'bar baz'])
print(shlex.split("'hello world' foo") == ['hello world', 'foo'])
print(shlex.split("''") == [''])

# ── split: double quotes ───────────────────────────────────────────────────────
print(shlex.split('"foo bar"') == ['foo bar'])
print(shlex.split('foo "bar baz"') == ['foo', 'bar baz'])
print(shlex.split('"hello world" foo') == ['hello world', 'foo'])
print(shlex.split('""') == [''])

# ── split: mixed quotes ────────────────────────────────────────────────────────
print(shlex.split("'foo' \"bar\"") == ['foo', 'bar'])
print(shlex.split("foo 'bar \"baz\"'") == ['foo', 'bar "baz"'])

# ── split: posix mode (default True) backslash ────────────────────────────────
print(shlex.split('foo\\ bar') == ['foo bar'])
print(shlex.split('foo\\nbar') == ['foonbar'])

# ── split: posix=False ────────────────────────────────────────────────────────
print(shlex.split('"foo bar"', posix=False) == ['"foo bar"'])
print(shlex.split("'foo bar'", posix=False) == ["'foo bar'"])

# ── split: comments ───────────────────────────────────────────────────────────
print(shlex.split('foo # comment', comments=True) == ['foo'])
print(shlex.split('foo # comment', comments=False) == ['foo', '#', 'comment'])
print(shlex.split('# all comment', comments=True) == [])

# ── split: return type ────────────────────────────────────────────────────────
result = shlex.split('a b c')
print(isinstance(result, list))
print(all(isinstance(x, str) for x in result))

# ── quote: safe strings pass through unchanged ────────────────────────────────
print(shlex.quote('foo') == 'foo')
print(shlex.quote('hello123') == 'hello123')
print(shlex.quote('foo-bar') == 'foo-bar')
print(shlex.quote('foo/bar') == 'foo/bar')
print(shlex.quote('foo_bar') == 'foo_bar')
print(shlex.quote('foo.bar') == 'foo.bar')
print(shlex.quote('foo:bar') == 'foo:bar')

# ── quote: unsafe strings get single-quoted ───────────────────────────────────
print(shlex.quote('') == "''")
print(shlex.quote('hello world') == "'hello world'")
print(shlex.quote('foo bar') == "'foo bar'")
print(shlex.quote('foo$bar') == "'foo$bar'")
print(shlex.quote('foo;bar') == "'foo;bar'")

# ── quote: single quote in string gets escaped ────────────────────────────────
print(shlex.quote("it's") == "'it'\"'\"'s'")

# ── quote: return type always str ─────────────────────────────────────────────
print(isinstance(shlex.quote('foo'), str))
print(isinstance(shlex.quote('foo bar'), str))

# ── join: basic ───────────────────────────────────────────────────────────────
print(shlex.join(['foo', 'bar', 'baz']) == 'foo bar baz')
print(shlex.join([]) == '')
print(shlex.join(['foo bar', 'baz']) == "'foo bar' baz")
print(shlex.join(['foo', 'bar baz']) == "foo 'bar baz'")

# ── join: round-trip with split ───────────────────────────────────────────────
tokens = ['echo', 'hello world', 'foo']
joined = shlex.join(tokens)
print(shlex.split(joined) == tokens)

tokens2 = ['ls', '-la', '/tmp/my dir']
joined2 = shlex.join(tokens2)
print(shlex.split(joined2) == tokens2)

# ── join: return type ─────────────────────────────────────────────────────────
print(isinstance(shlex.join(['foo', 'bar']), str))

# ── shlex class: instantiation ────────────────────────────────────────────────
print(hasattr(shlex, 'shlex'))
s = shlex.shlex()
print(isinstance(s, shlex.shlex))

# ── shlex class: default attribute values ─────────────────────────────────────
print(s.wordchars == 'abcdfeghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_')
print(s.whitespace == ' \t\r\n')
print(s.escape == '\\')
print(s.quotes == '\'"')
print(s.commenters == '#')
print(s.whitespace_split == False)
print(s.debug == 0)
print(s.lineno == 1)
print(s.token == '')
print(s.punctuation_chars == '')
print(s.posix == False)
print(s.eof == '')
print(s.source is None)
print(s.state == ' ')
print(hasattr(s, 'filestack'))
print(hasattr(s, 'pushback'))
print(s.infile is None)

# ── shlex class: posix=True ───────────────────────────────────────────────────
sp = shlex.shlex(posix=True)
print(sp.posix == True)

# ── shlex class: methods callable ─────────────────────────────────────────────
print(callable(s.get_token))
print(callable(s.read_token))
print(callable(s.push_token))
print(callable(s.push_source))
print(callable(s.pop_source))
print(callable(s.sourcehook))
print(callable(s.error_leader))

# ── shlex class: method return types ──────────────────────────────────────────
print(isinstance(s.get_token(), str))
print(isinstance(s.read_token(), str))
print(s.push_token('foo') is None)
print(s.push_source('dummy') is None)
print(s.pop_source() is None)
print(isinstance(s.error_leader(), str))

# ── shlex class: sourcehook returns tuple (or raises on real file open) ───────
try:
    hook = s.sourcehook('myfile')
    print(isinstance(hook, tuple))
    print(len(hook) == 2)
except (OSError, FileNotFoundError):
    print(True)
    print(True)

# ── split error on unclosed quote ─────────────────────────────────────────────
try:
    shlex.split("'unclosed")
    print(False)
except ValueError:
    print(True)

try:
    shlex.split('"unclosed')
    print(False)
except ValueError:
    print(True)

# ── callability checks ────────────────────────────────────────────────────────
print(callable(shlex.split))
print(callable(shlex.join))
print(callable(shlex.quote))

print('done')
