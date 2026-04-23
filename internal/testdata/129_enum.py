from enum import Enum, IntEnum, StrEnum, Flag, IntFlag, auto, unique

# ===== Basic Enum =====
class Color(Enum):
    RED = 1
    GREEN = 2
    BLUE = 3

print(Color.RED)              # Color.RED
print(repr(Color.RED))        # <Color.RED: 1>
print(Color.RED.name)         # RED
print(Color.RED.value)        # 1
print(Color(2))               # Color.GREEN
print(Color['BLUE'])          # Color.BLUE

for m in Color:
    print(m)

print(len(Color))             # 3
print(Color.RED in Color)     # True
print(list(Color.__members__))# ['RED', 'GREEN', 'BLUE']
print(Color.RED == Color.RED)   # True
print(Color.RED == Color.GREEN) # False
print(Color.RED is Color.RED)   # True

# ===== auto() for Enum =====
class Things(Enum):
    A = auto()
    B = auto()
    C = auto()

print(Things.A.value)  # 1
print(Things.B.value)  # 2
print(Things.C.value)  # 3

# ===== Aliases =====
class Shape(Enum):
    SQUARE = 2
    QUAD = 2    # alias

print(Shape.QUAD is Shape.SQUARE)  # True
print(list(Shape))                 # [<Shape.SQUARE: 2>]
print(len(Shape))                  # 1
print(list(Shape.__members__))     # ['SQUARE', 'QUAD']

# ===== @unique =====
try:
    @unique
    class Dupe(Enum):
        A = 1
        B = 1
    print('no error')
except ValueError:
    print('ValueError raised')

# ===== isinstance =====
print(isinstance(Color.RED, Enum))   # True
print(isinstance(Color.RED, Color))  # True
print(isinstance(1, Color))          # False

# ===== IntEnum =====
class Number(IntEnum):
    ONE = 1
    TWO = 2
    THREE = 3

print(Number.ONE)              # 1
print(repr(Number.ONE))        # <Number.ONE: 1>
print(Number.ONE == 1)         # True
print(Number.TWO > Number.ONE) # True
print(Number.ONE + 1)          # 2

# ===== StrEnum =====
class Mood(StrEnum):
    HAPPY = auto()
    SAD = 'sad'

print(Mood.HAPPY)              # happy
print(repr(Mood.HAPPY))        # <Mood.HAPPY: 'happy'>
print(str(Mood.HAPPY))         # happy
print(Mood.HAPPY == 'happy')   # True
print(Mood.SAD == 'sad')       # True

# ===== Flag =====
class Perm(Flag):
    READ = auto()
    WRITE = auto()
    EXECUTE = auto()

print(Perm.READ.value)         # 1
print(Perm.WRITE.value)        # 2
print(Perm.EXECUTE.value)      # 4
print(repr(Perm.READ))         # <Perm.READ: 1>

rw = Perm.READ | Perm.WRITE
print(rw.value)                # 3
print(repr(rw))                # <Perm.READ|WRITE: 3>
print(Perm.READ in rw)         # True
print(Perm.EXECUTE in rw)      # False

# ===== Functional API: list of names =====
Animal = Enum('Animal', ['ANT', 'BEE', 'CAT'])
print(Animal.ANT.value)        # 1
print(Animal.BEE.name)         # BEE
print(len(Animal))             # 3

# ===== Functional API: space-separated string =====
Direction = Enum('Direction', 'NORTH SOUTH EAST WEST')
print(Direction.NORTH.value)   # 1
print(Direction.WEST.value)    # 4

# ===== Functional API: dict =====
Status = Enum('Status', {'OK': 200, 'NOT_FOUND': 404})
print(Status.OK.value)         # 200
print(Status.NOT_FOUND.value)  # 404

print('done')
