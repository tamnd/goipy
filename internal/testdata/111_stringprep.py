import stringprep

# --- in_table_a1 ---
print(stringprep.in_table_a1('\u0378'))   # unassigned in Unicode 3.2
print(stringprep.in_table_a1('A'))        # assigned
print(stringprep.in_table_a1('\uFDD0'))   # FDD0-FDEF excluded from A.1
print(stringprep.in_table_a1('\uFFFE'))   # FFFE excluded

# --- in_table_b1 ---
print(stringprep.in_table_b1('\u00ad'))   # SOFT HYPHEN - True
print(stringprep.in_table_b1('\u034f'))   # COMBINING GRAPHEME JOINER - True
print(stringprep.in_table_b1('A'))        # False
print(stringprep.in_table_b1('\ufeff'))   # ZERO WIDTH NO-BREAK SPACE - True

# --- map_table_b2 ---
print(repr(stringprep.map_table_b2('A')))    # 'a'
print(repr(stringprep.map_table_b2('a')))    # 'a'
print(repr(stringprep.map_table_b2('\u00df')))  # 'ss' (sharp s)

# --- map_table_b3 ---
print(repr(stringprep.map_table_b3('A')))    # 'a'
print(repr(stringprep.map_table_b3('a')))    # 'a'
print(repr(stringprep.map_table_b3('\u00df')))  # 'ss'
print(repr(stringprep.map_table_b3('\u00b5')))  # '\u03bc' (mu)

# --- in_table_c11 ---
print(stringprep.in_table_c11(' '))      # True
print(stringprep.in_table_c11('a'))      # False
print(stringprep.in_table_c11('\u00a0')) # False (non-ASCII space)

# --- in_table_c12 ---
print(stringprep.in_table_c12('\u00a0')) # True (NO-BREAK SPACE)
print(stringprep.in_table_c12(' '))      # False
print(stringprep.in_table_c12('\u2000')) # True (EN QUAD)

# --- in_table_c11_c12 ---
print(stringprep.in_table_c11_c12(' '))      # True
print(stringprep.in_table_c11_c12('\u00a0')) # True
print(stringprep.in_table_c11_c12('a'))      # False

# --- in_table_c21 ---
print(stringprep.in_table_c21('\x00'))   # True
print(stringprep.in_table_c21('\x1f'))   # True
print(stringprep.in_table_c21('\x7f'))   # True
print(stringprep.in_table_c21('A'))      # False

# --- in_table_c22 ---
print(stringprep.in_table_c22('\x80'))   # True (non-ASCII Cc)
print(stringprep.in_table_c22('\u200c')) # True (in c22_specials)
print(stringprep.in_table_c22('\x00'))   # False (ASCII)
print(stringprep.in_table_c22('A'))      # False

# --- in_table_c21_c22 ---
print(stringprep.in_table_c21_c22('\x00'))   # True
print(stringprep.in_table_c21_c22('\x80'))   # True
print(stringprep.in_table_c21_c22('\u200c')) # True
print(stringprep.in_table_c21_c22('A'))      # False

# --- in_table_c3 ---
print(stringprep.in_table_c3('\ue000'))  # True (private use)
print(stringprep.in_table_c3('A'))       # False

# --- in_table_c4 ---
print(stringprep.in_table_c4('\ufdd0'))  # True (non-character)
print(stringprep.in_table_c4('\ufffe'))  # True
print(stringprep.in_table_c4('\uffff'))  # True
print(stringprep.in_table_c4('A'))       # False

# --- in_table_c5 ---
print(stringprep.in_table_c5('\ud800'))  # True (surrogate)
print(stringprep.in_table_c5('A'))       # False

# --- in_table_c6 ---
print(stringprep.in_table_c6('\ufff9'))  # True
print(stringprep.in_table_c6('\ufffd'))  # True
print(stringprep.in_table_c6('A'))       # False

# --- in_table_c7 ---
print(stringprep.in_table_c7('\u2ff0'))  # True
print(stringprep.in_table_c7('\u2ffb'))  # True
print(stringprep.in_table_c7('A'))       # False

# --- in_table_c8 ---
print(stringprep.in_table_c8('\u0340'))  # True
print(stringprep.in_table_c8('\u200e'))  # True
print(stringprep.in_table_c8('A'))       # False

# --- in_table_c9 ---
print(stringprep.in_table_c9('\U000e0001')) # True
print(stringprep.in_table_c9('\U000e0020')) # True
print(stringprep.in_table_c9('A'))          # False

# --- in_table_d1 ---
print(stringprep.in_table_d1('\u05be'))  # True (Hebrew, bidi R)
print(stringprep.in_table_d1('\u0627'))  # True (Arabic Alef, bidi AL)
print(stringprep.in_table_d1('A'))       # False

# --- in_table_d2 ---
print(stringprep.in_table_d2('A'))       # True (bidi L)
print(stringprep.in_table_d2('a'))       # True
print(stringprep.in_table_d2('\u0627'))  # False (bidi AL)
