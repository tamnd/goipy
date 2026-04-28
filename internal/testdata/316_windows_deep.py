import msvcrt
import winreg
import winsound
import msilib

# ── msvcrt constants ──────────────────────────────────────────────────────────
print(msvcrt.LK_UNLCK == 0)
print(msvcrt.LK_LOCK == 1)
print(msvcrt.LK_NBLCK == 2)
print(msvcrt.LK_RLCK == 3)
print(msvcrt.LK_NBRLCK == 4)
print(msvcrt.SEM_FAILCRITICALERRORS == 0x0001)
print(msvcrt.SEM_NOALIGNMENTFAULTEXCEPT == 0x0004)
print(msvcrt.SEM_NOGPFAULTERRORBOX == 0x0002)
print(msvcrt.SEM_NOOPENFILEERRORBOX == 0x8000)
print(isinstance(msvcrt.CRT_ASSEMBLY_VERSION, str))

# ── msvcrt callables ──────────────────────────────────────────────────────────
print(callable(msvcrt.locking))
print(callable(msvcrt.setmode))
print(callable(msvcrt.open_osfhandle))
print(callable(msvcrt.get_osfhandle))
print(callable(msvcrt.kbhit))
print(callable(msvcrt.getch))
print(callable(msvcrt.getwch))
print(callable(msvcrt.getche))
print(callable(msvcrt.getwche))
print(callable(msvcrt.putch))
print(callable(msvcrt.putwch))
print(callable(msvcrt.ungetch))
print(callable(msvcrt.ungetwch))
print(callable(msvcrt.heapmin))
print(callable(msvcrt.SetErrorMode))
print(callable(msvcrt.GetErrorMode))

# ── msvcrt stub return values ─────────────────────────────────────────────────
print(msvcrt.kbhit() == False)
print(isinstance(msvcrt.getch(), bytes))
print(isinstance(msvcrt.getwch(), str))
print(isinstance(msvcrt.setmode(0, 0), int))

# ── winsound constants ────────────────────────────────────────────────────────
print(winsound.SND_APPLICATION == 128)
print(winsound.SND_FILENAME == 131072)
print(winsound.SND_ALIAS == 65536)
print(winsound.SND_LOOP == 8)
print(winsound.SND_MEMORY == 4)
print(winsound.SND_PURGE == 64)
print(winsound.SND_ASYNC == 1)
print(winsound.SND_NODEFAULT == 2)
print(winsound.SND_NOSTOP == 16)
print(winsound.SND_NOWAIT == 8192)
print(winsound.SND_SENTRY == 524288)
print(winsound.SND_SYNC == 0)
print(winsound.SND_SYSTEM == 2097152)
print(winsound.MB_ICONASTERISK == 64)
print(winsound.MB_ICONEXCLAMATION == 48)
print(winsound.MB_ICONHAND == 16)
print(winsound.MB_ICONQUESTION == 32)
print(winsound.MB_OK == 0)
print(winsound.MB_ICONERROR == 16)
print(winsound.MB_ICONINFORMATION == 64)
print(winsound.MB_ICONSTOP == 16)
print(winsound.MB_ICONWARNING == 48)

# ── winsound callables ────────────────────────────────────────────────────────
print(callable(winsound.Beep))
print(callable(winsound.PlaySound))
print(callable(winsound.MessageBeep))

# ── winreg constants ──────────────────────────────────────────────────────────
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
print(winreg.HKEY_CLASSES_ROOT == 2147483648)
print(winreg.HKEY_CURRENT_USER == 2147483649)
print(winreg.HKEY_LOCAL_MACHINE == 2147483650)

# ── winreg error alias ────────────────────────────────────────────────────────
print(winreg.error is OSError)

# ── winreg HKEYType ───────────────────────────────────────────────────────────
print(isinstance(winreg.HKEYType, type))

# ── winreg callables ──────────────────────────────────────────────────────────
print(callable(winreg.CloseKey))
print(callable(winreg.ConnectRegistry))
print(callable(winreg.CreateKey))
print(callable(winreg.OpenKey))
print(callable(winreg.QueryValue))
print(callable(winreg.SetValueEx))
print(callable(winreg.ExpandEnvironmentStrings))

# ── msilib constants ──────────────────────────────────────────────────────────
print(msilib.AMD64 == False)
print(msilib.Win64 == False)
print(msilib.datasizemask == 0x00FF)
print(msilib.type_valid == 0x0100)
print(msilib.type_localizable == 0x0200)
print(msilib.typemask == 0x0C00)
print(msilib.type_long == 0x0000)
print(msilib.type_short == 0x0400)
print(msilib.type_string == 0x0C00)
print(msilib.type_binary == 0x0800)
print(msilib.type_nullable == 0x1000)
print(msilib.type_key == 0x2000)
print(msilib.knownbits == 0x3FFF)

# ── msilib classes ────────────────────────────────────────────────────────────
print(isinstance(msilib.Table, type))
print(isinstance(msilib.CAB, type))
print(isinstance(msilib.Directory, type))
print(isinstance(msilib.Binary, type))
print(isinstance(msilib.Feature, type))
print(isinstance(msilib.Control, type))
print(isinstance(msilib.RadioButtonGroup, type))
print(isinstance(msilib.Dialog, type))

# ── msilib callables ──────────────────────────────────────────────────────────
print(callable(msilib.make_id))
print(callable(msilib.gen_uuid))
print(callable(msilib.add_data))
print(callable(msilib.add_stream))
print(callable(msilib.init_database))
print(callable(msilib.add_tables))
print(callable(msilib.change_sequence))

# ── msilib gen_uuid returns str ───────────────────────────────────────────────
print(isinstance(msilib.gen_uuid(), str))

print('done')
