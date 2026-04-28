"""Comprehensive site module test — covers all public API from the Python docs."""
import site


# ── Constants ─────────────────────────────────────────────────────────────────

print(site.ENABLE_USER_SITE == True)          # True
print(isinstance(site.PREFIXES, list))        # True
print(all(isinstance(p, str) for p in site.PREFIXES))  # True
print(isinstance(site.USER_BASE, str))        # True
print(isinstance(site.USER_SITE, str))        # True
print(len(site.USER_BASE) > 0)               # True
print(len(site.USER_SITE) > 0)               # True


# ── getsitepackages ───────────────────────────────────────────────────────────

pkgs = site.getsitepackages()
print(isinstance(pkgs, list))                 # True
print(len(pkgs) > 0)                          # True
print(all(isinstance(p, str) for p in pkgs)) # True

pkgs2 = site.getsitepackages(["/tmp/p1"])
print(isinstance(pkgs2, list))               # True
print(all("/tmp/p1" in p for p in pkgs2))    # True


# ── getusersitepackages / getuserbase ─────────────────────────────────────────

usp = site.getusersitepackages()
print(isinstance(usp, str))                  # True
print(len(usp) > 0)                          # True

ub = site.getuserbase()
print(isinstance(ub, str))                   # True
print(len(ub) > 0)                           # True


# ── makepath ──────────────────────────────────────────────────────────────────

result = site.makepath("/usr", "lib", "python3")
print(isinstance(result, tuple))             # True
print(len(result) == 2)                      # True
print(isinstance(result[0], str))            # True
print(result[0] == result[1])                # True
print(result[0].endswith("python3"))         # True


# ── check_enableusersite ──────────────────────────────────────────────────────

print(site.check_enableusersite() == True)   # True


# ── gethistoryfile ────────────────────────────────────────────────────────────

hf = site.gethistoryfile()
print(isinstance(hf, str))                   # True
print(len(hf) > 0)                           # True
print("python_history" in hf)               # True


# ── removeduppaths ────────────────────────────────────────────────────────────

rdup = site.removeduppaths()
print(isinstance(rdup, set))                 # True


# ── addsitepackages ───────────────────────────────────────────────────────────

known = set()
result_sp = site.addsitepackages(known)
print(isinstance(result_sp, set))            # True

result_sp2 = site.addsitepackages(None)
print(result_sp2 is None or isinstance(result_sp2, set))  # True


# ── addusersitepackages ───────────────────────────────────────────────────────

result_usp = site.addusersitepackages(set())
print(isinstance(result_usp, set))           # True


# ── venv ──────────────────────────────────────────────────────────────────────

result_venv = site.venv(set())
print(isinstance(result_venv, set))          # True


# ── no-op stubs ───────────────────────────────────────────────────────────────

print(callable(site.addsitedir))             # True
print(callable(site.addpackage))             # True
print(callable(site.abs_paths))              # True
print(callable(site.enablerlcompleter))      # True
print(callable(site.execsitecustomize))      # True
print(callable(site.execusercustomize))      # True
print(callable(site.main))                   # True
print(callable(site.register_readline))      # True

site.addsitedir("/tmp/fake")
site.abs_paths()
site.enablerlcompleter()
site.register_readline()


# ── setquit ───────────────────────────────────────────────────────────────────

site.setquit()
import builtins
print(hasattr(builtins, 'quit'))             # True
print(hasattr(builtins, 'exit'))             # True
q = builtins.quit
print(type(q).__name__ == "Quitter")         # True
print("quit" in repr(q))                     # True
try:
    q()
except SystemExit:
    print(True)                              # True


# ── setcopyright ──────────────────────────────────────────────────────────────

site.setcopyright()
print(hasattr(builtins, 'copyright'))        # True
print(hasattr(builtins, 'credits'))          # True
print(hasattr(builtins, 'license'))          # True
print(type(builtins.copyright).__name__ == "_Printer")  # True
print(isinstance(repr(builtins.copyright), str))  # True


# ── sethelper ─────────────────────────────────────────────────────────────────

site.sethelper()
print(hasattr(builtins, 'help'))             # True
print(type(builtins.help).__name__ == "_Helper")  # True
print(isinstance(repr(builtins.help), str))  # True


print("done")
