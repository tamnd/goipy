import heapq

# ===== heappush / heappop =====
h = []
heapq.heappush(h, 5)
heapq.heappush(h, 3)
heapq.heappush(h, 7)
heapq.heappush(h, 1)
heapq.heappush(h, 4)
out = []
while h:
    out.append(heapq.heappop(h))
print(out)  # [1, 3, 4, 5, 7]

# single element push/pop
h = []
heapq.heappush(h, 42)
print(heapq.heappop(h))  # 42
print(h)  # []

# duplicates
h = []
for x in [3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5]:
    heapq.heappush(h, x)
out = []
while h:
    out.append(heapq.heappop(h))
print(out)  # [1, 1, 2, 3, 3, 4, 5, 5, 5, 6, 9]

# negative numbers
h = []
for x in [-3, -1, -4, -1, -5]:
    heapq.heappush(h, x)
out = []
while h:
    out.append(heapq.heappop(h))
print(out)  # [-5, -4, -3, -1, -1]

# strings
h = []
for s in ["banana", "apple", "cherry", "date"]:
    heapq.heappush(h, s)
out = []
while h:
    out.append(heapq.heappop(h))
print(out)  # ['apple', 'banana', 'cherry', 'date']

# ===== heappop empty raises IndexError =====
try:
    heapq.heappop([])
except IndexError:
    print("IndexError on empty heappop")

# ===== heapify =====
h = [5, 3, 8, 1, 9, 2, 7, 4]
heapq.heapify(h)
out = []
while h:
    out.append(heapq.heappop(h))
print(out)  # [1, 2, 3, 4, 5, 7, 8, 9]

# heapify already sorted
h = [1, 2, 3, 4, 5]
heapq.heapify(h)
print(heapq.heappop(h))  # 1

# heapify single element
h = [99]
heapq.heapify(h)
print(h[0])  # 99

# heapify empty
h = []
heapq.heapify(h)
print(h)  # []

# ===== heappushpop =====
# item <= heap[0]: return item without touching heap
h = [3, 5, 7]
heapq.heapify(h)
result = heapq.heappushpop(h, 1)
print(result)  # 1
print(sorted(h))  # [3, 5, 7]

# item > heap[0]: push item, pop minimum
h = [1, 5, 7]
heapq.heapify(h)
result = heapq.heappushpop(h, 4)
print(result)  # 1
print(sorted(h))  # [4, 5, 7]

# empty heap: just return item
h = []
result = heapq.heappushpop(h, 10)
print(result)  # 10
print(h)  # []

# ===== heapreplace =====
h = [1, 5, 9]
heapq.heapify(h)
old = heapq.heapreplace(h, 3)
print(old)  # 1
print(sorted(h))  # [3, 5, 9]

# replace with smaller value
h = [2, 6, 8]
heapq.heapify(h)
old = heapq.heapreplace(h, 1)
print(old)  # 2
print(sorted(h))  # [1, 6, 8]

# replace with larger value
h = [1, 3, 5]
heapq.heapify(h)
old = heapq.heapreplace(h, 10)
print(old)  # 1
print(sorted(h))  # [3, 5, 10]

# heapreplace on empty raises IndexError
try:
    heapq.heapreplace([], 1)
except IndexError:
    print("IndexError on empty heapreplace")

# ===== nlargest =====
nums = [3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5]
print(heapq.nlargest(3, nums))   # [9, 6, 5]
print(heapq.nlargest(1, nums))   # [9]
print(heapq.nlargest(0, nums))   # []
print(heapq.nlargest(20, nums))  # all 11, sorted desc

# nlargest with key=
data = [("alice", 30), ("bob", 25), ("carol", 35), ("dave", 28)]
top2 = heapq.nlargest(2, data, key=lambda x: x[1])
print([name for name, _ in top2])  # ['carol', 'alice']

# nlargest with negative key
print(heapq.nlargest(3, nums, key=lambda x: -x))  # [1, 1, 2]

# ===== nsmallest =====
print(heapq.nsmallest(3, nums))   # [1, 1, 2]
print(heapq.nsmallest(1, nums))   # [1]
print(heapq.nsmallest(0, nums))   # []
print(heapq.nsmallest(20, nums))  # all 11, sorted asc

# nsmallest with key=
bot2 = heapq.nsmallest(2, data, key=lambda x: x[1])
print([name for name, _ in bot2])  # ['bob', 'dave']

# nsmallest with string lengths
words = ["fig", "elderberry", "apple", "kiwi", "banana"]
print(heapq.nsmallest(3, words, key=len))  # ['fig', 'kiwi', 'apple']

# ===== merge (assumes sorted inputs) =====
# basic merge of sorted iterables
result = list(heapq.merge([1, 3, 5], [2, 4, 6]))
print(result)  # [1, 2, 3, 4, 5, 6]

# three iterables
result = list(heapq.merge([1, 4], [2, 5], [3, 6]))
print(result)  # [1, 2, 3, 4, 5, 6]

# empty iterables
result = list(heapq.merge([], [1, 2], []))
print(result)  # [1, 2]

# no iterables
result = list(heapq.merge())
print(result)  # []

# merge with reverse=True (inputs sorted descending)
result = list(heapq.merge([5, 3, 1], [6, 4, 2], reverse=True))
print(result)  # [6, 5, 4, 3, 2, 1]

# merge with key= (inputs already sorted by key)
words1 = ["c", "bb", "aaa"]     # sorted by length asc
words2 = ["ff", "eee", "dddd"]  # sorted by length asc
result = list(heapq.merge(words1, words2, key=len))
print(result)  # ['c', 'ff', 'bb', 'eee', 'aaa', 'dddd']

# ===== tuple-based priority queue (standard pattern) =====
tasks = []
heapq.heappush(tasks, (3, "low priority"))
heapq.heappush(tasks, (1, "high priority"))
heapq.heappush(tasks, (2, "medium priority"))
heapq.heappush(tasks, (1, "also high priority"))

out = []
while tasks:
    priority, task = heapq.heappop(tasks)
    out.append((priority, task))
print(out)

# ===== heap sort =====
def heapsort(iterable):
    h = []
    for v in iterable:
        heapq.heappush(h, v)
    return [heapq.heappop(h) for _ in range(len(h))]

print(heapsort([3, 1, 4, 1, 5, 9, 2, 6]))  # [1, 1, 2, 3, 4, 5, 6, 9]
print(heapsort([]))  # []
print(heapsort([42]))  # [42]

print('done')
