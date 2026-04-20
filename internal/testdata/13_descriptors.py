class Rect:
    def __init__(self, w, h):
        self._w = w
        self._h = h

    @property
    def area(self):
        return self._w * self._h

    @classmethod
    def square(cls, side):
        return cls(side, side)

    @staticmethod
    def describe():
        return "a rectangle"

r = Rect(3, 4)
print(r.area)
s = Rect.square(5)
print(s.area)
print(Rect.describe())
print(r.describe())
