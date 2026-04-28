import winreg

# ── HKEY root constants ───────────────────────────────────────────────────────
print(winreg.HKEY_CLASSES_ROOT == 2147483648)
print(winreg.HKEY_CURRENT_USER == 2147483649)
print(winreg.HKEY_LOCAL_MACHINE == 2147483650)
print(winreg.HKEY_USERS == 2147483651)
print(winreg.HKEY_PERFORMANCE_DATA == 2147483652)
print(winreg.HKEY_CURRENT_CONFIG == 2147483653)
print(winreg.HKEY_DYN_DATA == 2147483654)
print(isinstance(winreg.HKEY_CLASSES_ROOT, int))
print(isinstance(winreg.HKEY_LOCAL_MACHINE, int))

# ── KEY access constants ──────────────────────────────────────────────────────
print(winreg.KEY_ALL_ACCESS == 983103)
print(winreg.KEY_WRITE == 131078)
print(winreg.KEY_READ == 131097)
print(winreg.KEY_EXECUTE == 131097)
print(winreg.KEY_QUERY_VALUE == 1)
print(winreg.KEY_SET_VALUE == 2)
print(winreg.KEY_CREATE_SUB_KEY == 4)
print(winreg.KEY_ENUMERATE_SUB_KEYS == 8)
print(winreg.KEY_NOTIFY == 16)
print(winreg.KEY_CREATE_LINK == 32)
print(winreg.KEY_WOW64_64KEY == 256)
print(winreg.KEY_WOW64_32KEY == 512)

# ── REG value type constants (documented) ────────────────────────────────────
print(winreg.REG_NONE == 0)
print(winreg.REG_SZ == 1)
print(winreg.REG_EXPAND_SZ == 2)
print(winreg.REG_BINARY == 3)
print(winreg.REG_DWORD == 4)
print(winreg.REG_DWORD_LITTLE_ENDIAN == 4)
print(winreg.REG_DWORD_BIG_ENDIAN == 5)
print(winreg.REG_LINK == 6)
print(winreg.REG_MULTI_SZ == 7)
print(winreg.REG_RESOURCE_LIST == 8)
print(winreg.REG_FULL_RESOURCE_DESCRIPTOR == 9)
print(winreg.REG_RESOURCE_REQUIREMENTS_LIST == 10)
print(winreg.REG_QWORD == 11)
print(winreg.REG_QWORD_LITTLE_ENDIAN == 11)

# ── REG undocumented constants ────────────────────────────────────────────────
print(winreg.REG_CREATED_NEW_KEY == 1)
print(winreg.REG_LEGAL_CHANGE_FILTER == 268435471)
print(winreg.REG_LEGAL_OPTION == 31)
print(winreg.REG_NOTIFY_CHANGE_ATTRIBUTES == 2)
print(winreg.REG_NOTIFY_CHANGE_LAST_SET == 4)
print(winreg.REG_NOTIFY_CHANGE_NAME == 1)
print(winreg.REG_NOTIFY_CHANGE_SECURITY == 8)
print(winreg.REG_NO_LAZY_FLUSH == 4)
print(winreg.REG_OPENED_EXISTING_KEY == 2)
print(winreg.REG_OPTION_BACKUP_RESTORE == 4)
print(winreg.REG_OPTION_CREATE_LINK == 2)
print(winreg.REG_OPTION_NON_VOLATILE == 0)
print(winreg.REG_OPTION_OPEN_LINK == 8)
print(winreg.REG_OPTION_RESERVED == 0)
print(winreg.REG_OPTION_VOLATILE == 1)
print(winreg.REG_REFRESH_HIVE == 2)
print(winreg.REG_WHOLE_HIVE_VOLATILE == 1)

# ── error alias ───────────────────────────────────────────────────────────────
print(winreg.error is OSError)

# ── HKEYType is a type ────────────────────────────────────────────────────────
print(isinstance(winreg.HKEYType, type))

# ── HKEYType instance methods and attributes ──────────────────────────────────
h = winreg.HKEYType()
print(isinstance(h, winreg.HKEYType))
print(isinstance(h.handle, int))
print(h.handle == 0)
print(h.Close() is None)
print(isinstance(h.Detach(), int))
print(h.Detach() == 0)

# ── HKEYType __bool__ and __int__ ────────────────────────────────────────────
print(isinstance(bool(h), bool))
print(isinstance(int(h), int))

# ── HKEYType context manager ─────────────────────────────────────────────────
with winreg.HKEYType() as k:
    print(isinstance(k, winreg.HKEYType))

# ── all registry functions callable ─────────────────────────────────────────
print(callable(winreg.CloseKey))
print(callable(winreg.ConnectRegistry))
print(callable(winreg.CreateKey))
print(callable(winreg.CreateKeyEx))
print(callable(winreg.DeleteKey))
print(callable(winreg.DeleteKeyEx))
print(callable(winreg.DeleteValue))
print(callable(winreg.EnumKey))
print(callable(winreg.EnumValue))
print(callable(winreg.ExpandEnvironmentStrings))
print(callable(winreg.FlushKey))
print(callable(winreg.LoadKey))
print(callable(winreg.OpenKey))
print(callable(winreg.OpenKeyEx))
print(callable(winreg.QueryInfoKey))
print(callable(winreg.QueryValue))
print(callable(winreg.QueryValueEx))
print(callable(winreg.SaveKey))
print(callable(winreg.SetValue))
print(callable(winreg.SetValueEx))
print(callable(winreg.DisableReflectionKey))
print(callable(winreg.EnableReflectionKey))
print(callable(winreg.QueryReflectionKey))

# ── registry functions raise OSError when called ─────────────────────────────
try:
    winreg.OpenKey(winreg.HKEY_LOCAL_MACHINE, 'SOFTWARE')
    print(False)
except OSError:
    print(True)

try:
    winreg.QueryValue(winreg.HKEY_CURRENT_USER, None)
    print(False)
except OSError:
    print(True)

try:
    winreg.CreateKey(winreg.HKEY_LOCAL_MACHINE, 'Test')
    print(False)
except OSError:
    print(True)

print('done')
