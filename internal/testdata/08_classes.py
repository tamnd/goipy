class Point:
    def __init__(self, x, y):
        self.x = x
        self.y = y
    def dist2(self):
        return self.x * self.x + self.y * self.y
    def __repr__(self):
        return "Point(" + str(self.x) + "," + str(self.y) + ")"

p = Point(3, 4)
print(p.x, p.y)
print(p.dist2())

class Animal:
    def __init__(self, name):
        self.name = name
    def speak(self):
        return "..."

class Dog(Animal):
    def speak(self):
        return self.name + " says woof"

d = Dog("Rex")
print(d.speak())
print(isinstance(d, Dog))
print(isinstance(d, Animal))
