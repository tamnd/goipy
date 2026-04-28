import winsound

# ── SND_* playback flag constants ─────────────────────────────────────────────
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

# ── SND_* added in 3.14 ───────────────────────────────────────────────────────
print(winsound.SND_SENTRY == 524288)
print(winsound.SND_SYNC == 0)
print(winsound.SND_SYSTEM == 2097152)

# ── all SND_* are ints ────────────────────────────────────────────────────────
print(isinstance(winsound.SND_APPLICATION, int))
print(isinstance(winsound.SND_FILENAME, int))
print(isinstance(winsound.SND_ALIAS, int))
print(isinstance(winsound.SND_LOOP, int))
print(isinstance(winsound.SND_MEMORY, int))
print(isinstance(winsound.SND_PURGE, int))
print(isinstance(winsound.SND_ASYNC, int))
print(isinstance(winsound.SND_NODEFAULT, int))
print(isinstance(winsound.SND_NOSTOP, int))
print(isinstance(winsound.SND_NOWAIT, int))
print(isinstance(winsound.SND_SENTRY, int))
print(isinstance(winsound.SND_SYNC, int))
print(isinstance(winsound.SND_SYSTEM, int))

# ── MB_* MessageBeep type constants ──────────────────────────────────────────
print(winsound.MB_ICONASTERISK == 64)
print(winsound.MB_ICONEXCLAMATION == 48)
print(winsound.MB_ICONHAND == 16)
print(winsound.MB_ICONQUESTION == 32)
print(winsound.MB_OK == 0)

# ── MB_* added in 3.14 ───────────────────────────────────────────────────────
print(winsound.MB_ICONERROR == 16)
print(winsound.MB_ICONINFORMATION == 64)
print(winsound.MB_ICONSTOP == 16)
print(winsound.MB_ICONWARNING == 48)

# ── alias relationships ───────────────────────────────────────────────────────
print(winsound.MB_ICONERROR == winsound.MB_ICONHAND)
print(winsound.MB_ICONINFORMATION == winsound.MB_ICONASTERISK)
print(winsound.MB_ICONSTOP == winsound.MB_ICONHAND)
print(winsound.MB_ICONWARNING == winsound.MB_ICONEXCLAMATION)

# ── all MB_* are ints ─────────────────────────────────────────────────────────
print(isinstance(winsound.MB_ICONASTERISK, int))
print(isinstance(winsound.MB_ICONEXCLAMATION, int))
print(isinstance(winsound.MB_ICONHAND, int))
print(isinstance(winsound.MB_ICONQUESTION, int))
print(isinstance(winsound.MB_OK, int))
print(isinstance(winsound.MB_ICONERROR, int))
print(isinstance(winsound.MB_ICONINFORMATION, int))
print(isinstance(winsound.MB_ICONSTOP, int))
print(isinstance(winsound.MB_ICONWARNING, int))

# ── SND_* flags can be ORed ──────────────────────────────────────────────────
combined = winsound.SND_FILENAME | winsound.SND_ASYNC
print(isinstance(combined, int))
print(combined == 131073)

combined2 = winsound.SND_ALIAS | winsound.SND_LOOP | winsound.SND_ASYNC
print(isinstance(combined2, int))
print(combined2 == 65545)

# ── functions callable ────────────────────────────────────────────────────────
print(callable(winsound.Beep))
print(callable(winsound.PlaySound))
print(callable(winsound.MessageBeep))

# ── function return values ────────────────────────────────────────────────────
print(winsound.Beep(440, 100) is None)
print(winsound.PlaySound(None, winsound.SND_PURGE) is None)
print(winsound.PlaySound('SystemDefault', winsound.SND_ALIAS | winsound.SND_ASYNC) is None)
print(winsound.MessageBeep() is None)
print(winsound.MessageBeep(winsound.MB_OK) is None)
print(winsound.MessageBeep(winsound.MB_ICONHAND) is None)

print('done')
