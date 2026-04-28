import importlib.metadata as im

# ── 1. API surface ─────────────────────────────────────────────────────────────
print(callable(im.version))
print(callable(im.metadata))
print(callable(im.requires))
print(callable(im.files))
print(callable(im.distribution))
print(callable(im.distributions))
print(callable(im.packages_distributions))
print(callable(im.entry_points))
print(callable(im.PackageNotFoundError))
print(callable(im.Distribution))
print(callable(im.PathDistribution))
print(callable(im.EntryPoint))
print(callable(im.EntryPoints))
print(callable(im.PackagePath))
print(callable(im.PackageMetadata))

# ── 2. PackageNotFoundError ────────────────────────────────────────────────────
print(issubclass(im.PackageNotFoundError, ImportError))

try:
    im.version('no-such-package-xyz-abc')
except im.PackageNotFoundError:
    print('caught PackageNotFoundError')

# ── 3. version() ───────────────────────────────────────────────────────────────
v = im.version('testpkgmeta')
print(v)

# ── 4. distribution() ──────────────────────────────────────────────────────────
d = im.distribution('testpkgmeta')
print(d.name)
print(d.version)
print(type(d).__name__)

# ── 5. metadata() ──────────────────────────────────────────────────────────────
meta = im.metadata('testpkgmeta')
print(type(meta).__name__)
print(meta['Name'])
print(meta['Version'])
print(meta['License'])
print('Requires-Dist' in meta)
print('Nonexistent-Key' in meta)

reqs = meta.get_all('Requires-Dist')
print(type(reqs).__name__)
print(len(reqs))

keys = meta.keys()
print('Name' in [str(k) for k in keys])

print(meta.get('Nonexistent-Key', 'default_val'))

# ── 6. requires() ──────────────────────────────────────────────────────────────
reqs2 = im.requires('testpkgmeta')
print(type(reqs2).__name__)
print(len(reqs2))
print('urllib3>=1.0' in reqs2)

# ── 7. files() ─────────────────────────────────────────────────────────────────
fls = im.files('testpkgmeta')
print(fls is not None)
if fls is not None:
    print(type(fls).__name__)
    print(len(fls) > 0)
    print(type(fls[0]).__name__)

# ── 8. entry_points() — use [plugins] group unique to testpkgmeta ──────────────
all_eps = im.entry_points()
print(type(all_eps).__name__)

# [plugins] group only exists in testpkgmeta so count is deterministic
eps_pl = im.entry_points(group='plugins')
print(type(eps_pl).__name__)
print(len(eps_pl))
pl_names = sorted(e.name for e in eps_pl)
print(pl_names)

# console_scripts: find testpkgmeta-cli specifically
eps_cs = im.entry_points(group='console_scripts')
cli_eps = eps_cs.select(name='testpkgmeta-cli')
print(len(cli_eps))
ep = cli_eps[0]
print(ep.name)
print(ep.group)
print(ep.value)

# ── 9. packages_distributions() ───────────────────────────────────────────────
pd = im.packages_distributions()
print(type(pd).__name__)
print('testpkgmeta' in pd)
print(type(pd['testpkgmeta']).__name__)

# ── 10. distributions() ────────────────────────────────────────────────────────
dists = list(im.distributions())
print(type(dists).__name__)
dist_names = [d.name for d in dists]
print('testpkgmeta' in dist_names)

# ── 11. EntryPoint construction ────────────────────────────────────────────────
ep2 = im.EntryPoint(name='myapp', group='console_scripts', value='myapp:main')
print(ep2.name)
print(ep2.group)
print(ep2.value)

# ── 12. EntryPoints construction and select() ──────────────────────────────────
ep3 = im.EntryPoint(name='myapp2', group='gui_scripts', value='myapp2:run')
eps2 = im.EntryPoints([ep2, ep3])
print(type(eps2).__name__)
print(len(eps2))

selected = eps2.select(group='console_scripts')
print(type(selected).__name__)
print(len(selected))
print(selected[0].name)

selected2 = eps2.select(group='gui_scripts')
print(len(selected2))
print(selected2[0].name)

selected3 = eps2.select(group='nonexistent')
print(len(selected3))

# ── 13. Distribution.from_name ─────────────────────────────────────────────────
d2 = im.Distribution.from_name('testpkgmeta')
print(d2.name)
print(d2.version)

try:
    im.Distribution.from_name('no-such-package-xyz')
except im.PackageNotFoundError:
    print('from_name raised PackageNotFoundError')

# ── 14. PathDistribution.read_text ─────────────────────────────────────────────
txt = d.read_text('top_level.txt')
print(txt.strip())

print('done')
