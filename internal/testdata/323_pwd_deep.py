import pwd

# ── struct_passwd class exists ─────────────────────────────────────────────────
assert hasattr(pwd, 'struct_passwd')

# ── getpwuid(0) returns root entry ─────────────────────────────────────────────
r = pwd.getpwuid(0)
assert r.pw_name == 'root'
assert isinstance(r.pw_name, str)
assert isinstance(r.pw_passwd, str)
assert r.pw_uid == 0
assert isinstance(r.pw_uid, int)
assert isinstance(r.pw_gid, int)
assert isinstance(r.pw_gecos, str)
assert isinstance(r.pw_dir, str)
assert isinstance(r.pw_shell, str)
assert r.pw_dir != ''
assert r.pw_shell != ''

# ── getpwuid unknown uid raises KeyError ───────────────────────────────────────
try:
    pwd.getpwuid(99999999)
    raised = False
except KeyError:
    raised = True
assert raised

# ── getpwnam('root') returns root entry ───────────────────────────────────────
r2 = pwd.getpwnam('root')
assert r2.pw_name == 'root'
assert r2.pw_uid == 0
assert isinstance(r2.pw_uid, int)
assert isinstance(r2.pw_gid, int)
assert isinstance(r2.pw_dir, str)
assert isinstance(r2.pw_shell, str)

# ── getpwnam unknown name raises KeyError ─────────────────────────────────────
try:
    pwd.getpwnam('__nonexistent_user_xyz__')
    raised = False
except KeyError:
    raised = True
assert raised

# ── getpwall returns non-empty list ───────────────────────────────────────────
all_entries = pwd.getpwall()
assert isinstance(all_entries, list)
assert len(all_entries) >= 1

# each entry is a struct_passwd with all fields
first = all_entries[0]
assert isinstance(first.pw_name, str)
assert isinstance(first.pw_uid, int)
assert isinstance(first.pw_gid, int)
assert isinstance(first.pw_dir, str)
assert isinstance(first.pw_shell, str)

# ── uid round-trip: getpwuid(getpwnam('root').pw_uid).pw_name == 'root' ────────
uid = pwd.getpwnam('root').pw_uid
assert pwd.getpwuid(uid).pw_name == 'root'

# ── callable checks ────────────────────────────────────────────────────────────
assert callable(pwd.getpwuid)
assert callable(pwd.getpwnam)
assert callable(pwd.getpwall)

print('ok')
