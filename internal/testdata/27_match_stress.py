class Point:
    __match_args__ = ("x", "y")
    def __init__(self, x, y):
        self.x = x
        self.y = y

class Circle:
    __match_args__ = ("center", "radius")
    def __init__(self, center, radius):
        self.center = center
        self.radius = radius


def describe(shape):
    match shape:
        case Point(0, 0):
            return "origin"
        case Point(0, y):
            return f"y-axis at {y}"
        case Point(x, 0):
            return f"x-axis at {x}"
        case Point(x, y) if x == y:
            return f"diagonal {x}"
        case Point(x, y):
            return f"point {x},{y}"
        case Circle(Point(0, 0), radius=r):
            return f"origin circle r={r}"
        case Circle(Point(x, y), r):
            return f"circle@({x},{y}) r={r}"
        case _:
            return "?"


for s in [
    Point(0, 0),
    Point(0, 5),
    Point(5, 0),
    Point(3, 3),
    Point(1, 2),
    Circle(Point(0, 0), 10),
    Circle(Point(4, 5), 2),
    42,
]:
    print(describe(s))


def kind(v):
    match v:
        case 1 | 2 | 3:
            return "small"
        case int() if v > 100:
            return "big"
        case int():
            return "mid"
        case [1, 2, *rest] if len(rest) > 0:
            return f"list12+{rest}"
        case {"tag": "x", **more}:
            return f"tag-x more={sorted(more.items())}"
        case _:
            return "other"


for v in [2, 150, 50, [1, 2, 3, 4], {"tag": "x", "a": 1, "b": 2}, {"tag": "y"}, "hi"]:
    print(kind(v))
