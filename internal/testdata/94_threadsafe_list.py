# List operations from the Python 3.13+ thread safety docs.

lst = [10, 20, 30, 40, 50]

# --- Atomic reads ---
print(lst[2])            # 30
print(20 in lst)         # True
print(lst.index(40))     # 3
print(lst.count(20))     # 1

# --- Atomic write ---
lst[0] = 99
print(lst[0])            # 99

# --- New-object ops ---
a = [1, 2, 3]
b = [4, 5, 6]
c = a + b
print(c)                 # [1, 2, 3, 4, 5, 6]
print(2 * a)             # [1, 2, 1, 2, 1, 2] — wait, 2*a = a repeated twice
print(a.copy())          # [1, 2, 3]

# --- End-only ---
lst2 = [1, 2, 3]
lst2.append(4)
print(lst2)              # [1, 2, 3, 4]
print(lst2.pop())        # 4
print(lst2)              # [1, 2, 3]

# --- clear / sort / reverse ---
lst3 = [3, 1, 2]
lst3.sort()
print(lst3)              # [1, 2, 3]
lst3.reverse()
print(lst3)              # [3, 2, 1]
lst3.clear()
print(lst3)              # []

# --- Shifting ops ---
lst4 = [1, 2, 3, 4]
lst4.insert(1, 99)
print(lst4)              # [1, 99, 2, 3, 4]
print(lst4.pop(2))       # 2
print(lst4)              # [1, 99, 3, 4]
lst4 *= 2
print(lst4)              # [1, 99, 3, 4, 1, 99, 3, 4]

# --- remove / extend / slice-assign ---
lst5 = [1, 2, 3, 4]
lst5.remove(3)
print(lst5)              # [1, 2, 4]
lst5.extend([5, 6])
print(lst5)              # [1, 2, 4, 5, 6]
lst5[1:3] = [20, 40]
print(lst5)              # [1, 20, 40, 5, 6]

# --- Safe patterns ---
# copy-before-iterate
shared = [1, 2, 3, 4, 5]
snapshot = shared.copy()
total = 0
for x in snapshot:
    total += x
print(total)             # 15

# if-then-pop (safe only at tail)
q = [1, 2, 3]
if q:
    print(q.pop())       # 3
print(q)                 # [1, 2]
