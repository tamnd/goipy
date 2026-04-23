import itertools
import operator

# ===== count =====
print(list(itertools.islice(itertools.count(), 5)))          # [0, 1, 2, 3, 4]
print(list(itertools.islice(itertools.count(10), 4)))        # [10, 11, 12, 13]
print(list(itertools.islice(itertools.count(1, 2), 5)))      # [1, 3, 5, 7, 9]
print(list(itertools.islice(itertools.count(0, -1), 4)))     # [0, -1, -2, -3]
# float step
vals = list(itertools.islice(itertools.count(1.0, 0.5), 4))
print([round(v, 1) for v in vals])                          # [1.0, 1.5, 2.0, 2.5]

# ===== cycle =====
print(list(itertools.islice(itertools.cycle('AB'), 5)))      # ['A', 'B', 'A', 'B', 'A']
print(list(itertools.islice(itertools.cycle([1, 2, 3]), 7))) # [1, 2, 3, 1, 2, 3, 1]
# empty cycle produces nothing
print(list(itertools.islice(itertools.cycle([]), 5)))        # []

# ===== repeat =====
print(list(itertools.repeat(7, 3)))                          # [7, 7, 7]
print(list(itertools.repeat('x', 0)))                       # []
print(list(itertools.islice(itertools.repeat(None), 3)))    # [None, None, None]

# ===== chain =====
print(list(itertools.chain([1, 2], [3, 4], [5])))           # [1, 2, 3, 4, 5]
print(list(itertools.chain('AB', 'CD')))                    # ['A', 'B', 'C', 'D']
print(list(itertools.chain()))                              # []
print(list(itertools.chain([1])))                           # [1]

# ===== chain.from_iterable =====
print(list(itertools.chain.from_iterable([[1,2],[3],[4,5]])))  # [1, 2, 3, 4, 5]
print(list(itertools.chain.from_iterable('ABCD')))             # ['A', 'B', 'C', 'D']
print(list(itertools.chain.from_iterable([])))                 # []

# ===== compress =====
print(list(itertools.compress('ABCDEF', [1,0,1,0,1,1])))    # ['A', 'C', 'E', 'F']
print(list(itertools.compress([1,2,3], [True, False, True])))# [1, 3]
print(list(itertools.compress('ABC', [0,0,0])))              # []
print(list(itertools.compress('AB', [1,1,1])))               # ['A', 'B'] (shorter data wins)

# ===== dropwhile =====
print(list(itertools.dropwhile(lambda x: x < 3, [1,2,3,4,1])))  # [3, 4, 1]
print(list(itertools.dropwhile(lambda x: x < 0, [1,2,3])))      # [1, 2, 3]
print(list(itertools.dropwhile(lambda x: True, [1,2,3])))        # []

# ===== takewhile =====
print(list(itertools.takewhile(lambda x: x < 3, [1,2,3,4,1])))  # [1, 2]
print(list(itertools.takewhile(lambda x: True, [1,2,3])))        # [1, 2, 3]
print(list(itertools.takewhile(lambda x: False, [1,2,3])))       # []

# ===== islice =====
print(list(itertools.islice('ABCDEFG', 4)))           # ['A', 'B', 'C', 'D']
print(list(itertools.islice('ABCDEFG', 2, 4)))        # ['C', 'D']
print(list(itertools.islice('ABCDEFG', 2, None)))     # ['C', 'D', 'E', 'F', 'G']
print(list(itertools.islice('ABCDEFG', 0, None, 2))) # ['A', 'C', 'E', 'G']
print(list(itertools.islice('ABCDEFG', 0)))           # []
print(list(itertools.islice(range(10), 2, 8, 3)))    # [2, 5]

# ===== starmap =====
print(list(itertools.starmap(pow, [(2,5),(3,2),(10,3)])))   # [32, 9, 1000]
print(list(itertools.starmap(operator.add, [(1,2),(3,4)]))) # [3, 7]
print(list(itertools.starmap(max, [(1,3),(2,1)])))          # [3, 2]

# ===== filterfalse =====
print(list(itertools.filterfalse(lambda x: x%2==0, range(8))))  # [1, 3, 5, 7]
print(list(itertools.filterfalse(None, [0,1,'',2,'a',False])))  # [0, '', False]
print(list(itertools.filterfalse(lambda x: True, [1,2,3])))     # []

# ===== groupby =====
# groups consecutive equal keys
data = [1,1,2,2,2,3,1,1]
result = [(k, list(g)) for k, g in itertools.groupby(data)]
print(result)  # [(1, [1, 1]), (2, [2, 2, 2]), (3, [3]), (1, [1, 1])]

# with key fn
words = ['apple','ant','ball','boy','cat']
result = [(k, list(g)) for k, g in itertools.groupby(words, key=lambda w: w[0])]
print(result)  # [('a', ['apple', 'ant']), ('b', ['ball', 'boy']), ('c', ['cat'])]

# empty
print(list(itertools.groupby([])))  # []

# ===== pairwise =====
print(list(itertools.pairwise('ABCDE')))         # [('A','B'),('B','C'),('C','D'),('D','E')]
print(list(itertools.pairwise([1,2,3])))          # [(1,2),(2,3)]
print(list(itertools.pairwise('A')))              # []
print(list(itertools.pairwise([])))               # []

# ===== tee =====
it1, it2 = itertools.tee([1,2,3])
print(list(it1))   # [1, 2, 3]
print(list(it2))   # [1, 2, 3]

it1, it2, it3 = itertools.tee([1,2,3], 3)
print(list(it1) == list(it2) == [1, 2, 3])  # True (it3 consumed)

print(list(itertools.tee([], 2)[0]))         # []

# ===== zip_longest =====
print(list(itertools.zip_longest([1,2,3], 'ab')))              # [(1,'a'),(2,'b'),(3,None)]
print(list(itertools.zip_longest([1,2], [3,4,5], fillvalue=0))) # [(1,3),(2,4),(0,5)]
print(list(itertools.zip_longest('AB', 'xy', fillvalue='-')))  # [('A','x'),('B','y')]
print(list(itertools.zip_longest()))                            # []
print(list(itertools.zip_longest([1])))                        # [(1,)]

# ===== accumulate =====
print(list(itertools.accumulate([1,2,3,4,5])))                           # [1,3,6,10,15]
print(list(itertools.accumulate([1,2,3,4,5], operator.mul)))             # [1,2,6,24,120]
print(list(itertools.accumulate([1,2,3,4,5], max)))                      # [1,2,3,4,5]
print(list(itertools.accumulate([3,1,4,1,5], min)))                      # [3,1,1,1,1]
print(list(itertools.accumulate([1,2,3], initial=0)))                    # [0,1,3,6]
print(list(itertools.accumulate([1,2,3], operator.mul, initial=1)))      # [1,1,2,6]
print(list(itertools.accumulate([])))                                     # []
print(list(itertools.accumulate([], initial=100)))                       # [100]

# ===== batched =====
print(list(itertools.batched('ABCDEFG', 3)))   # [('A','B','C'),('D','E','F'),('G',)]
print(list(itertools.batched(range(6), 2)))    # [(0,1),(2,3),(4,5)]
print(list(itertools.batched([], 3)))           # []
print(list(itertools.batched([1], 3)))          # [(1,)]
# strict=False (default): incomplete final batch allowed
print(list(itertools.batched('ABCDE', 3, strict=False)))  # [('A','B','C'),('D','E')]
# strict=True: incomplete batch raises ValueError
try:
    list(itertools.batched('ABCDE', 3, strict=True))
except ValueError:
    print('strict ok')                          # strict ok

# ===== product =====
print(list(itertools.product([1,2], 'ab')))          # [(1,'a'),(1,'b'),(2,'a'),(2,'b')]
print(list(itertools.product([1,2], repeat=2)))      # [(1,1),(1,2),(2,1),(2,2)]
print(list(itertools.product('AB', repeat=2)))       # [('A','A'),('A','B'),('B','A'),('B','B')]
print(list(itertools.product([], [1,2])))            # []
print(list(itertools.product()))                     # [()]
print(list(itertools.product([1])))                  # [(1,)]

# ===== permutations =====
print(list(itertools.permutations([1,2,3], 2)))    # [(1,2),(1,3),(2,1),(2,3),(3,1),(3,2)]
print(list(itertools.permutations('AB')))           # [('A','B'),('B','A')]
print(len(list(itertools.permutations([1,2,3,4])))) # 24
print(list(itertools.permutations([], 0)))           # [()]
print(list(itertools.permutations([1,2], 0)))        # [()]
print(list(itertools.permutations([1,2,3], 4)))      # [] (r > n)

# ===== combinations =====
print(list(itertools.combinations([1,2,3,4], 2)))   # [(1,2),(1,3),(1,4),(2,3),(2,4),(3,4)]
print(list(itertools.combinations('ABC', 2)))        # [('A','B'),('A','C'),('B','C')]
print(list(itertools.combinations([1,2,3], 0)))      # [()]
print(list(itertools.combinations([1,2,3], 4)))      # []  (r > n)
print(len(list(itertools.combinations(range(5),3)))) # 10

# ===== combinations_with_replacement =====
print(list(itertools.combinations_with_replacement([1,2,3], 2)))
# [(1,1),(1,2),(1,3),(2,2),(2,3),(3,3)]
print(list(itertools.combinations_with_replacement('AB', 3)))
# [('A','A','A'),('A','A','B'),('A','B','B'),('B','B','B')]
print(list(itertools.combinations_with_replacement([1], 3)))  # [(1,1,1)]
print(list(itertools.combinations_with_replacement([1,2], 0))) # [()]

# ===== edge cases =====
# islice with step > 1
print(list(itertools.islice(range(100), 0, 10, 3)))  # [0, 3, 6, 9]

# accumulate with lambda
print(list(itertools.accumulate([1,2,3,4], lambda a,b: a-b)))  # [1,-1,-4,-8]

# starmap with zip
pairs = list(zip([1,2,3],[4,5,6]))
print(list(itertools.starmap(lambda a,b: a+b, pairs)))  # [5, 7, 9]

# chain.from_iterable with generator-like
print(list(itertools.chain.from_iterable([[i]*i for i in range(1,4)])))  # [1,2,2,3,3,3]

# tee n=1
(only_one,) = itertools.tee([1,2,3], 1)
print(list(only_one))  # [1, 2, 3]

# product repeat=0 -> [()]
print(list(itertools.product([1,2,3], repeat=0)))  # [()]

# count with keyword-like usage via islice
print(list(itertools.islice(itertools.count(5, -2), 4)))  # [5, 3, 1, -1]

print('done')
