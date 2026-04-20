class Color:
    RED = "red"
    GREEN = "green"
    BLUE = "blue"


class Node:
    __match_args__ = ("tag", "kids")
    def __init__(self, tag, kids=None):
        self.tag = tag
        self.kids = kids or []


def label(c):
    match c:
        case Color.RED:
            return "R"
        case Color.GREEN:
            return "G"
        case Color.BLUE:
            return "B"
        case _:
            return "?"


for c in ["red", "green", "blue", "orange"]:
    print(label(c))


def tree_depth(n):
    match n:
        case Node(_, []):
            return 1
        case Node(_, [k]):
            return 1 + tree_depth(k)
        case Node(_, kids):
            return 1 + max(tree_depth(k) for k in kids) if False else 1 + max_depth(kids)
        case _:
            return 0


def max_depth(kids):
    best = 0
    for k in kids:
        d = tree_depth(k)
        if d > best:
            best = d
    return best


root = Node("a", [
    Node("b", [Node("c", [Node("d")])]),
    Node("e"),
    Node("f", [Node("g"), Node("h")]),
])
print(tree_depth(root))


def classify(v):
    match v:
        case [1, 2] | [3, 4]:
            return "pair-ab"
        case [x, y] | (x, y) if x == y:
            return f"twin {x}"
        case [*_, last]:
            return f"last {last}"
        case {"k": 1 | 2 | 3 as k}:
            return f"k-small {k}"
        case {"k": int() as k, **rest} if rest:
            return f"k={k} extras"
        case Node(tag="leaf"):
            return "leaf node"
        case Node(tag, kids) if len(kids) > 2:
            return f"wide {tag}"
        case _:
            return "other"


cases = [
    [1, 2],
    [3, 4],
    (7, 7),
    [9, 9],
    [10, 20, 30, 99],
    {"k": 2},
    {"k": 5, "extra": True},
    Node("leaf"),
    Node("root", [Node("x"), Node("y"), Node("z"), Node("w")]),
    "no match",
]
for v in cases:
    print(classify(v))


def deep(v):
    match v:
        case {"op": "add", "args": [int() as a, int() as b]}:
            return a + b
        case {"op": "neg", "args": [int() as a]}:
            return -a
        case {"op": "apply", "args": [{"name": str() as n}, *rest]}:
            return f"{n}({len(rest)})"
        case _:
            return None


print(deep({"op": "add", "args": [2, 3]}))
print(deep({"op": "neg", "args": [9]}))
print(deep({"op": "apply", "args": [{"name": "f"}, 1, 2, 3]}))
print(deep({"op": "nope"}))
