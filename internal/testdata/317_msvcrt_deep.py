import msvcrt

# ── LK_* locking mode constants ───────────────────────────────────────────────
print(msvcrt.LK_UNLCK == 0)
print(msvcrt.LK_LOCK == 1)
print(msvcrt.LK_NBLCK == 2)
print(msvcrt.LK_RLCK == 3)
print(msvcrt.LK_NBRLCK == 4)
print(isinstance(msvcrt.LK_UNLCK, int))
print(isinstance(msvcrt.LK_LOCK, int))
print(isinstance(msvcrt.LK_NBLCK, int))
print(isinstance(msvcrt.LK_RLCK, int))
print(isinstance(msvcrt.LK_NBRLCK, int))

# ── SEM_* error mode constants ────────────────────────────────────────────────
print(msvcrt.SEM_FAILCRITICALERRORS == 1)
print(msvcrt.SEM_NOALIGNMENTFAULTEXCEPT == 4)
print(msvcrt.SEM_NOGPFAULTERRORBOX == 2)
print(msvcrt.SEM_NOOPENFILEERRORBOX == 32768)
print(isinstance(msvcrt.SEM_FAILCRITICALERRORS, int))
print(isinstance(msvcrt.SEM_NOALIGNMENTFAULTEXCEPT, int))
print(isinstance(msvcrt.SEM_NOGPFAULTERRORBOX, int))
print(isinstance(msvcrt.SEM_NOOPENFILEERRORBOX, int))

# ── CRT_ASSEMBLY_VERSION ──────────────────────────────────────────────────────
print(isinstance(msvcrt.CRT_ASSEMBLY_VERSION, str))
print(len(msvcrt.CRT_ASSEMBLY_VERSION) > 0)

# ── file operations are callable ─────────────────────────────────────────────
print(callable(msvcrt.locking))
print(callable(msvcrt.setmode))
print(callable(msvcrt.open_osfhandle))
print(callable(msvcrt.get_osfhandle))

# ── file operation return types ───────────────────────────────────────────────
print(msvcrt.locking(0, 0, 0) is None)
print(isinstance(msvcrt.setmode(0, 0), int))
print(isinstance(msvcrt.open_osfhandle(0, 0), int))
print(isinstance(msvcrt.get_osfhandle(0), int))

# ── console I/O -- callable ───────────────────────────────────────────────────
print(callable(msvcrt.kbhit))
print(callable(msvcrt.getch))
print(callable(msvcrt.getche))
print(callable(msvcrt.getwch))
print(callable(msvcrt.getwche))
print(callable(msvcrt.putch))
print(callable(msvcrt.putwch))
print(callable(msvcrt.ungetch))
print(callable(msvcrt.ungetwch))

# ── console I/O -- return types ───────────────────────────────────────────────
print(isinstance(msvcrt.kbhit(), bool))
print(msvcrt.kbhit() == False)
print(isinstance(msvcrt.getch(), bytes))
print(isinstance(msvcrt.getche(), bytes))
print(isinstance(msvcrt.getwch(), str))
print(isinstance(msvcrt.getwche(), str))
print(msvcrt.putch(b'a') is None)
print(msvcrt.putwch('a') is None)
print(msvcrt.ungetch(b'a') is None)
print(msvcrt.ungetwch('a') is None)

# ── heapmin ───────────────────────────────────────────────────────────────────
print(callable(msvcrt.heapmin))
print(msvcrt.heapmin() is None)

# ── SetErrorMode / GetErrorMode ───────────────────────────────────────────────
print(callable(msvcrt.SetErrorMode))
print(callable(msvcrt.GetErrorMode))
print(isinstance(msvcrt.SetErrorMode(0), int))
print(isinstance(msvcrt.GetErrorMode(), int))

# ── SEM_* values can be ORed together ────────────────────────────────────────
combined = msvcrt.SEM_FAILCRITICALERRORS | msvcrt.SEM_NOGPFAULTERRORBOX
print(isinstance(combined, int))
print(combined == 3)

print('done')
