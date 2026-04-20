def classify(p):
    match p:
        case 0:
            return "zero"
        case int(n) if n < 0:
            return f"neg {n}"
        case [a, b]:
            return f"pair {a},{b}"
        case [a, *rest]:
            return f"list {a} rest={rest}"
        case {"op": op, "v": v}:
            return f"map {op}={v}"
        case str() as s:
            return f"str {s!r}"
        case _:
            return "other"

for p in [0, -3, [1, 2], [1, 2, 3, 4], {"op": "add", "v": 7}, "hi", 3.14]:
    print(classify(p))
