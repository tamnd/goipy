# Module whose VALUE can be observed changing between two reload() calls.
# The test mutates the source file on disk between the reloads — this
# module is only the initial snapshot.
VALUE = "v1"

def read():
    return VALUE
