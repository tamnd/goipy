import sysconfig

# get_python_version
v = sysconfig.get_python_version()
print(v == "3.14")  # True

# get_platform returns a non-empty string
p = sysconfig.get_platform()
print(len(p) > 0)   # True

# get_scheme_names returns a tuple with known entries
names = sysconfig.get_scheme_names()
print(isinstance(names, tuple))   # True
print("posix_prefix" in names or "nt" in names)  # True

# get_default_scheme returns a string
ds = sysconfig.get_default_scheme()
print(isinstance(ds, str))  # True

# get_paths returns a dict with required keys
paths = sysconfig.get_paths()
print(isinstance(paths, dict))   # True
print("stdlib" in paths)         # True
print("purelib" in paths)        # True
print("scripts" in paths)        # True

# get_path for a known key
stdlib = sysconfig.get_path("stdlib")
print(isinstance(stdlib, str))   # True

# get_path_names
pnames = sysconfig.get_path_names()
print("stdlib" in pnames)   # True
print("scripts" in pnames)  # True

# get_config_vars returns a dict
cvars = sysconfig.get_config_vars()
print(isinstance(cvars, dict))   # True
print("prefix" in cvars)         # True

# get_config_var for a key
pv = sysconfig.get_config_var("prefix")
print(isinstance(pv, str))  # True

# get_config_vars with args returns a list
result = sysconfig.get_config_vars("prefix", "exec_prefix")
print(isinstance(result, list))   # True
print(len(result) == 2)           # True

print("done")
