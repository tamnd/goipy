import tokenize
import io

# ── 1. Re-exported constants ──────────────────────────────────────────────────
print(tokenize.ENDMARKER == 0)
print(tokenize.NAME == 1)
print(tokenize.NUMBER == 2)
print(tokenize.STRING == 3)
print(tokenize.NEWLINE == 4)
print(tokenize.INDENT == 5)
print(tokenize.DEDENT == 6)
print(tokenize.OP == 55)
print(tokenize.COMMENT == 65)
print(tokenize.NL == 66)
print(tokenize.ERRORTOKEN == 67)
print(tokenize.ENCODING == 68)
print(tokenize.N_TOKENS == 69)
print(tokenize.NT_OFFSET == 256)

# ── 2. BOM_UTF8 and tabsize ───────────────────────────────────────────────────
print(tokenize.BOM_UTF8 == b'\xef\xbb\xbf')
print(tokenize.tabsize == 8)

# ── 3. Pattern strings are str ────────────────────────────────────────────────
print(isinstance(tokenize.Whitespace, str))
print(isinstance(tokenize.Comment, str))
print(isinstance(tokenize.Name, str))
print(isinstance(tokenize.Number, str))
print(isinstance(tokenize.Funny, str))

# ── 4. TokenError is subclass of Exception ────────────────────────────────────
print(issubclass(tokenize.TokenError, Exception))

# ── 5. TokenInfo construction ─────────────────────────────────────────────────
ti = tokenize.TokenInfo(1, 'hello', (1, 0), (1, 5), 'hello\n')
print(ti.type == 1)
print(ti.string == 'hello')
print(ti.start == (1, 0))
print(ti.end == (1, 5))
print(ti.line == 'hello\n')

# ── 6. TokenInfo.exact_type ───────────────────────────────────────────────────
# NAME token: exact_type == type
name_ti = tokenize.TokenInfo(1, 'foo', (1, 0), (1, 3), 'foo\n')
print(name_ti.exact_type == 1)

# OP token '+': exact_type == PLUS (14)
op_plus = tokenize.TokenInfo(55, '+', (1, 0), (1, 1), '+\n')
print(op_plus.exact_type == 14)

# OP token '==': exact_type == EQEQUAL (27)
op_eq = tokenize.TokenInfo(55, '==', (1, 0), (1, 2), '==\n')
print(op_eq.exact_type == 27)

# OP token '(': exact_type == LPAR (7)
op_lpar = tokenize.TokenInfo(55, '(', (1, 0), (1, 1), '(\n')
print(op_lpar.exact_type == 7)

# NUMBER token: exact_type == type (2)
num_ti = tokenize.TokenInfo(2, '42', (1, 0), (1, 2), '42\n')
print(num_ti.exact_type == 2)

# ── 7. tokenize() on b'x = 1\n' ──────────────────────────────────────────────
src = b'x = 1\n'
tokens = list(tokenize.tokenize(io.BytesIO(src).readline))
types = [t.type for t in tokens]
print(tokenize.ENCODING in types)
print(tokenize.NAME in types)
print(tokenize.OP in types)
print(tokenize.NUMBER in types)
print(tokenize.NEWLINE in types)
print(tokenize.ENDMARKER in types)

# Check specific token content
enc_tok = tokens[0]
print(enc_tok.type == tokenize.ENCODING)
print(enc_tok.string == 'utf-8')

name_tok = tokens[1]
print(name_tok.type == tokenize.NAME)
print(name_tok.string == 'x')

# ── 8. generate_tokens() on 'x = 1\n' ────────────────────────────────────────
src_str = 'x = 1\n'
gtokens = list(tokenize.generate_tokens(io.StringIO(src_str).readline))
gtypes = [t.type for t in gtokens]
print(tokenize.NAME in gtypes)
print(tokenize.OP in gtypes)
print(tokenize.NUMBER in gtypes)
print(tokenize.ENDMARKER in gtypes)
# No ENCODING token in generate_tokens
print(tokenize.ENCODING not in gtypes)

# ── 9. tokenize() on comment line ────────────────────────────────────────────
src2 = b'# hello\n'
tokens2 = list(tokenize.tokenize(io.BytesIO(src2).readline))
types2 = [t.type for t in tokens2]
print(tokenize.COMMENT in types2)

# ── 10. detect_encoding() defaults to utf-8 ───────────────────────────────────
enc, lines = tokenize.detect_encoding(io.BytesIO(b'x = 1\n').readline)
print(enc == 'utf-8')

# detect_encoding with explicit coding declaration
coding_src = b'# coding: utf-8\nx = 1\n'
enc2, lines2 = tokenize.detect_encoding(io.BytesIO(coding_src).readline)
print(enc2 == 'utf-8')

# ── 11. detect_encoding with BOM ──────────────────────────────────────────────
bom_src = b'\xef\xbb\xbfx = 1\n'
enc3, _ = tokenize.detect_encoding(io.BytesIO(bom_src).readline)
print(enc3 == 'utf-8-sig')

# ── 12. untokenize() with 2-tuples ────────────────────────────────────────────
toks2 = [(tokenize.NAME, 'x'), (tokenize.OP, '='), (tokenize.NUMBER, '1'), (tokenize.NEWLINE, '\n')]
result = tokenize.untokenize(toks2)
print(isinstance(result, str))
print('x' in result)
print('=' in result)
print('1' in result)

# ── 13. Untokenizer class ─────────────────────────────────────────────────────
ut = tokenize.Untokenizer()
print(ut is not None)

# ── 14. tok_name and EXACT_TOKEN_TYPES re-exported ───────────────────────────
print(isinstance(tokenize.tok_name, dict))
print(tokenize.tok_name[0] == 'ENDMARKER')
print(tokenize.tok_name[1] == 'NAME')
print(isinstance(tokenize.EXACT_TOKEN_TYPES, dict))
print(tokenize.EXACT_TOKEN_TYPES['+'] == 14)
print(tokenize.EXACT_TOKEN_TYPES['=='] == 27)

# ── 15. ISEOF/ISTERMINAL/ISNONTERMINAL re-exported ───────────────────────────
print(tokenize.ISEOF(0) == True)
print(tokenize.ISEOF(1) == False)
print(tokenize.ISTERMINAL(1) == True)
print(tokenize.ISTERMINAL(256) == False)
print(tokenize.ISNONTERMINAL(256) == True)
print(tokenize.ISNONTERMINAL(1) == False)

print('done')
