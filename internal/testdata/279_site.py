import site

# ENABLE_USER_SITE
print(site.ENABLE_USER_SITE == True)   # True

# USER_BASE and USER_SITE are strings
print(isinstance(site.USER_BASE, str))   # True
print(isinstance(site.USER_SITE, str))   # True
print(len(site.USER_BASE) > 0)           # True
print(len(site.USER_SITE) > 0)           # True

# getsitepackages returns a non-empty list of strings
pkgs = site.getsitepackages()
print(isinstance(pkgs, list))     # True
print(len(pkgs) > 0)              # True
print(all(isinstance(p, str) for p in pkgs))  # True

# getusersitepackages returns a string
usp = site.getusersitepackages()
print(isinstance(usp, str))       # True
print(len(usp) > 0)               # True

# getuserbase returns a string
ub = site.getuserbase()
print(isinstance(ub, str))        # True
print(len(ub) > 0)                # True

# addsitedir is a no-op
site.addsitedir("/tmp/fake")

# PREFIXES is a list
print(isinstance(site.PREFIXES, list))  # True

print("done")
