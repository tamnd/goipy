# Second helper for 37_import_stress. Module-level code sets up a small
# table and keeps track of how many times it was executed (the import
# system must run the module body at most once per process).

RUN_COUNT = [0]
RUN_COUNT[0] += 1

TABLE = {i: i * i for i in range(5)}

def total():
    return sum(TABLE.values())

def get(k, default=None):
    return TABLE.get(k, default)

class Box:
    def __init__(self, v):
        self.value = v
    def unwrap(self):
        return self.value
