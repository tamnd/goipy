import textwrap

# --- dedent ---
print(repr(textwrap.dedent("    line1\n    line2\n")))
print(repr(textwrap.dedent("\n    hello\n    world\n    ")))
# Mixed indentation: strip common prefix
print(repr(textwrap.dedent("    a\n  b\n")))

# --- indent ---
print(textwrap.indent("hello\nworld\n", "  "))
print(textwrap.indent("hello\n\nworld\n", "> "))     # blank lines not indented
# predicate: all lines
print(textwrap.indent("hello\nworld\n", "# ", predicate=lambda l: True))
# predicate: only non-blank
print(textwrap.indent("hello\n\nworld\n", "> ", predicate=lambda l: l.strip()))
# predicate: only blank
print(textwrap.indent("hello\n\nworld\n", "> ", predicate=lambda l: not l.strip()))

# --- wrap ---
text = "The quick brown fox jumped over the lazy dog"

print(textwrap.wrap(text, 15))
print(textwrap.wrap(text, 15, initial_indent="  "))
print(textwrap.wrap(text, 15, subsequent_indent="  "))

# max_lines
print(textwrap.wrap(text, 15, max_lines=2))
print(textwrap.wrap(text, 15, max_lines=2, placeholder=' ...'))

# empty / short
print(textwrap.wrap("", 10))
print(textwrap.wrap("hi", 10))

# long word breaks
print(textwrap.wrap("superlongword", 5))

# expand_tabs
print(textwrap.wrap("\thello world", 15, expand_tabs=True, tabsize=4))

# --- fill ---
print(textwrap.fill(text, 20))
print(textwrap.fill(text, 20, initial_indent="  "))
print(textwrap.fill("", 10))

# --- shorten ---
print(textwrap.shorten("Hello world", 12))
print(textwrap.shorten("Hello world", 5))
print(textwrap.shorten("Hello   world", 10))
print(textwrap.shorten("Hello world this is long", 15, placeholder="..."))
print(textwrap.shorten("", 10))

# --- TextWrapper ---
tw = textwrap.TextWrapper(width=15)
print(tw.wrap(text))
print(tw.fill(text))
print(tw.width)

tw2 = textwrap.TextWrapper(width=20, initial_indent="  ", subsequent_indent="    ")
print(tw2.wrap(text))
print(tw2.initial_indent)
print(tw2.subsequent_indent)

# max_lines on TextWrapper
tw3 = textwrap.TextWrapper(width=15, max_lines=2, placeholder=" ...")
print(tw3.wrap(text))
