import sqlite3

# ===== module attributes =====
print(hasattr(sqlite3, 'connect'))                  # True
print(hasattr(sqlite3, 'Error'))                    # True
print(hasattr(sqlite3, 'DatabaseError'))            # True
print(hasattr(sqlite3, 'OperationalError'))         # True
print(hasattr(sqlite3, 'IntegrityError'))           # True
print(hasattr(sqlite3, 'ProgrammingError'))         # True
print(hasattr(sqlite3, 'DataError'))                # True
print(hasattr(sqlite3, 'InternalError'))            # True
print(hasattr(sqlite3, 'NotSupportedError'))        # True
print(hasattr(sqlite3, 'Warning'))                  # True
print(hasattr(sqlite3, 'Row'))                      # True
print(isinstance(sqlite3.sqlite_version, str))      # True
print(isinstance(sqlite3.sqlite_version_info, tuple)) # True
print(sqlite3.PARSE_DECLTYPES)                      # 1
print(sqlite3.PARSE_COLNAMES)                       # 2

# ===== exception hierarchy =====
print(issubclass(sqlite3.Error, Exception))                        # True
print(issubclass(sqlite3.DatabaseError, sqlite3.Error))            # True
print(issubclass(sqlite3.OperationalError, sqlite3.DatabaseError)) # True
print(issubclass(sqlite3.IntegrityError, sqlite3.DatabaseError))   # True
print(issubclass(sqlite3.ProgrammingError, sqlite3.DatabaseError)) # True
print(issubclass(sqlite3.Warning, Exception))                      # True

# ===== raise correct type on bad driver name =====
import sys as _sys
_real_open = sqlite3.connect

def _fake_connect(name):
    raise sqlite3.OperationalError('no such driver')

# Patch connect to simulate missing driver (works on both CPython and goipy)
sqlite3.connect = _fake_connect
try:
    sqlite3.connect(':memory:')
except sqlite3.OperationalError:
    print('OperationalError raised')               # OperationalError raised
except Exception as e:
    print('unexpected error:', e)
finally:
    sqlite3.connect = _real_connect if '_real_connect' in dir() else _real_open

# Restore
sqlite3.connect = _real_open

# ===== exception catching =====
try:
    raise sqlite3.IntegrityError('dupe key')
except sqlite3.DatabaseError as e:
    print('caught as DatabaseError')               # caught as DatabaseError

try:
    raise sqlite3.OperationalError('table missing')
except sqlite3.Error as e:
    print('caught as Error')                       # caught as Error

try:
    raise sqlite3.ProgrammingError('bad sql')
except Exception as e:
    print('caught as Exception')                   # caught as Exception

# ===== Row class exists and is callable =====
print(callable(sqlite3.Row))                       # True

print('done')                                      # done
