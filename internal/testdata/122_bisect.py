import bisect

# ===== bisect_left basics =====
a = [1, 3, 5, 7, 9]
print(bisect.bisect_left(a, 5))    # 2  — exact match, leftmost
print(bisect.bisect_left(a, 4))    # 2  — between 3 and 5
print(bisect.bisect_left(a, 0))    # 0  — before all
print(bisect.bisect_left(a, 10))   # 5  — after all
print(bisect.bisect_left(a, 1))    # 0  — exact match at start
print(bisect.bisect_left(a, 9))    # 4  — exact match at end

# ===== bisect_right basics =====
print(bisect.bisect_right(a, 5))   # 3  — exact match, rightmost
print(bisect.bisect_right(a, 4))   # 2  — between 3 and 5
print(bisect.bisect_right(a, 0))   # 0  — before all
print(bisect.bisect_right(a, 10))  # 5  — after all
print(bisect.bisect_right(a, 1))   # 1  — after first element
print(bisect.bisect_right(a, 9))   # 5  — after last element

# bisect is an alias for bisect_right
print(bisect.bisect(a, 5))         # 3
print(bisect.bisect(a, 4))         # 2

# ===== duplicates: left vs right =====
dup = [1, 1, 2, 3, 3, 3, 4, 5, 5]
print(bisect.bisect_left(dup, 1))   # 0
print(bisect.bisect_right(dup, 1))  # 2
print(bisect.bisect_left(dup, 3))   # 3
print(bisect.bisect_right(dup, 3))  # 6
print(bisect.bisect_left(dup, 5))   # 7
print(bisect.bisect_right(dup, 5))  # 9

# ===== all equal =====
eq = [7, 7, 7, 7]
print(bisect.bisect_left(eq, 7))    # 0
print(bisect.bisect_right(eq, 7))   # 4
print(bisect.bisect_left(eq, 6))    # 0
print(bisect.bisect_right(eq, 8))   # 4

# ===== empty list =====
print(bisect.bisect_left([], 5))    # 0
print(bisect.bisect_right([], 5))   # 0

# ===== single element =====
print(bisect.bisect_left([3], 3))   # 0
print(bisect.bisect_right([3], 3))  # 1
print(bisect.bisect_left([3], 2))   # 0
print(bisect.bisect_left([3], 4))   # 1

# ===== lo / hi bounds =====
a = [1, 3, 5, 7, 9, 11]
# restrict search to [1:4] → sublist is [3, 5, 7]
print(bisect.bisect_left(a, 5, lo=1, hi=4))    # 2
print(bisect.bisect_right(a, 5, lo=1, hi=4))   # 3
print(bisect.bisect_left(a, 4, 1, 4))          # 2  — positional lo/hi
print(bisect.bisect_right(a, 4, 1, 4))         # 2
# lo == hi → always return lo
print(bisect.bisect_left(a, 5, lo=3, hi=3))    # 3
print(bisect.bisect_right(a, 5, lo=3, hi=3))   # 3
# lo at end
print(bisect.bisect_left(a, 5, lo=6))          # 6
# hi at start
print(bisect.bisect_right(a, 5, hi=0))         # 0

# ===== key= parameter =====
# search by second element in tuples
records = [('alice', 25), ('bob', 30), ('carol', 35), ('dave', 40)]
# records sorted by age; needle is the age value
print(bisect.bisect_left(records, 30, key=lambda r: r[1]))   # 1
print(bisect.bisect_right(records, 30, key=lambda r: r[1]))  # 2
print(bisect.bisect_left(records, 27, key=lambda r: r[1]))   # 1
print(bisect.bisect_right(records, 27, key=lambda r: r[1]))  # 1

# key= with string lengths
words = ['a', 'bb', 'ccc', 'dddd', 'eeeee']  # sorted by length
print(bisect.bisect_left(words, 3, key=len))    # 2
print(bisect.bisect_right(words, 3, key=len))   # 3
print(bisect.bisect_left(words, 0, key=len))    # 0
print(bisect.bisect_right(words, 6, key=len))   # 5

# key= with negation (descending list)
desc = [9, 7, 5, 3, 1]
# treat as sorted by negative value
print(bisect.bisect_left(desc, -5, key=lambda x: -x))   # 2
print(bisect.bisect_right(desc, -5, key=lambda x: -x))  # 3

# ===== insort_left =====
b = [1, 3, 5, 7]
bisect.insort_left(b, 4)
print(b)   # [1, 3, 4, 5, 7]

b = [1, 3, 3, 5]
bisect.insort_left(b, 3)
print(b)   # [1, 3, 3, 3, 5] — inserted before existing 3s

b = []
bisect.insort_left(b, 1)
print(b)   # [1]

bisect.insort_left(b, 0)
print(b)   # [0, 1]

bisect.insort_left(b, 2)
print(b)   # [0, 1, 2]

# ===== insort_right =====
b = [1, 3, 5, 7]
bisect.insort_right(b, 4)
print(b)   # [1, 3, 4, 5, 7]

b = [1, 3, 3, 5]
bisect.insort_right(b, 3)
print(b)   # [1, 3, 3, 3, 5] — inserted after existing 3s

# insort is alias for insort_right
b = [2, 4, 6]
bisect.insort(b, 3)
print(b)   # [2, 3, 4, 6]

b = [1, 3, 3, 5]
bisect.insort(b, 3)
print(b)   # [1, 3, 3, 3, 5]

# ===== insort with lo/hi =====
b = [1, 2, 3, 10, 20, 30]
bisect.insort(b, 15, lo=3, hi=6)
print(b)   # [1, 2, 3, 10, 15, 20, 30]

# ===== insort with key= =====
# insert by second field
items = [('a', 1), ('c', 3), ('e', 5)]
bisect.insort(items, ('d', 4), key=lambda r: r[1])
print(items)   # [('a', 1), ('c', 3), ('d', 4), ('e', 5)]

bisect.insort_left(items, ('b', 2), key=lambda r: r[1])
print(items)   # [('a', 1), ('b', 2), ('c', 3), ('d', 4), ('e', 5)]

# ===== documented usage patterns =====

# Grade lookup
def grade(score, breakpoints=[60, 70, 80, 90], grades='FDCBA'):
    i = bisect.bisect(breakpoints, score)
    return grades[i]

print(grade(33))    # F
print(grade(60))    # D
print(grade(70))    # C
print(grade(87))    # B
print(grade(99))    # A

# Count items <= x using bisect_right
def count_le(a, x):
    return bisect.bisect_right(a, x)

data = [1, 2, 2, 3, 4, 5, 5, 5, 6]
print(count_le(data, 2))   # 3  (1, 2, 2)
print(count_le(data, 5))   # 8  (all but 6)
print(count_le(data, 0))   # 0
print(count_le(data, 99))  # 9

# Find leftmost item >= x
def find_ge(a, x):
    i = bisect.bisect_left(a, x)
    if i < len(a):
        return a[i]
    raise ValueError

sorted_list = [1, 3, 5, 7, 9]
print(find_ge(sorted_list, 4))   # 5
print(find_ge(sorted_list, 5))   # 5
print(find_ge(sorted_list, 1))   # 1

# Find leftmost value > x
def find_gt(a, x):
    i = bisect.bisect_right(a, x)
    if i < len(a):
        return a[i]
    raise ValueError

print(find_gt(sorted_list, 4))   # 5
print(find_gt(sorted_list, 5))   # 7
print(find_gt(sorted_list, 1))   # 3

# ===== strings =====
words = ['apple', 'banana', 'cherry', 'date', 'elderberry']
print(bisect.bisect_left(words, 'cherry'))    # 2
print(bisect.bisect_right(words, 'cherry'))   # 3
print(bisect.bisect_left(words, 'blueberry')) # 2
print(bisect.bisect_right(words, 'fig'))      # 5

b = ['apple', 'cherry', 'elderberry']
bisect.insort(b, 'banana')
bisect.insort(b, 'date')
print(b)  # ['apple', 'banana', 'cherry', 'date', 'elderberry']

print('done')
