"""Test class-level annotations and annotated assignment behavior."""


class Point:
    def __init__(self, x, y):
        self.x = x
        self.y = y
    label: str = "origin"


p = Point(3, 4)
print(p.x, p.y, p.label)


class Counter:
    count: int = 0

    def increment(self):
        self.count += 1
        return self.count


c = Counter()
print(c.count)
c.count = 5
print(c.count)
print(c.increment())


def greet(name: str, times: int = 1) -> str:
    return (name + " ") * times


print(greet("hello", 3).strip())
print(greet("hi"))


class Animal:
    sound: str = "..."

    def speak(self):
        return self.sound


class Dog(Animal):
    sound: str = "woof"


d = Dog()
print(d.speak())
print(d.sound)
