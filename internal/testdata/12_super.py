class A:
    def greet(self):
        return "A"
    def kind(self):
        return "animal"

class B(A):
    def greet(self):
        return super().greet() + "B"

class C(B):
    def greet(self):
        return super().greet() + "C"
    def kind(self):
        return super().kind() + "/mammal"

print(A().greet())
print(B().greet())
print(C().greet())
print(C().kind())
