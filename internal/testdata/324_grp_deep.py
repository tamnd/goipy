import grp

# ── struct_group class exists ──────────────────────────────────────────────────
assert hasattr(grp, 'struct_group')

# ── getgrgid(0) returns wheel entry ───────────────────────────────────────────
r = grp.getgrgid(0)
assert r.gr_name == 'wheel'
assert isinstance(r.gr_name, str)
assert isinstance(r.gr_passwd, str)
assert r.gr_gid == 0
assert isinstance(r.gr_gid, int)
assert isinstance(r.gr_mem, list)
assert 'root' in r.gr_mem

# ── getgrgid unknown gid raises KeyError ──────────────────────────────────────
try:
    grp.getgrgid(99999999)
    raised = False
except KeyError:
    raised = True
assert raised

# ── getgrnam('wheel') returns wheel entry ─────────────────────────────────────
r2 = grp.getgrnam('wheel')
assert r2.gr_name == 'wheel'
assert r2.gr_gid == 0
assert isinstance(r2.gr_gid, int)
assert isinstance(r2.gr_mem, list)

# ── getgrnam unknown name raises KeyError ─────────────────────────────────────
try:
    grp.getgrnam('__nonexistent_group_xyz__')
    raised = False
except KeyError:
    raised = True
assert raised

# ── getgrall returns non-empty list ───────────────────────────────────────────
all_entries = grp.getgrall()
assert isinstance(all_entries, list)
assert len(all_entries) >= 1

# each entry is a struct_group with all fields
first = all_entries[0]
assert isinstance(first.gr_name, str)
assert isinstance(first.gr_gid, int)
assert isinstance(first.gr_mem, list)

# ── gid round-trip ─────────────────────────────────────────────────────────────
gid = grp.getgrnam('wheel').gr_gid
assert grp.getgrgid(gid).gr_name == 'wheel'

# ── name round-trip ────────────────────────────────────────────────────────────
name = grp.getgrgid(0).gr_name
assert grp.getgrnam(name).gr_gid == 0

# ── callable checks ────────────────────────────────────────────────────────────
assert callable(grp.getgrgid)
assert callable(grp.getgrnam)
assert callable(grp.getgrall)

print('ok')
