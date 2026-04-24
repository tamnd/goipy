"""Tests for getpass module."""
import getpass

# getpass() returns a string (non-interactive: empty string)
result = getpass.getpass()
print(isinstance(result, str))    # True

# getpass() with positional prompt
result2 = getpass.getpass('PIN: ')
print(isinstance(result2, str))   # True

# getpass() with keyword prompt
result3 = getpass.getpass(prompt='Enter key: ')
print(isinstance(result3, str))   # True

# getuser() returns a non-empty string
user = getpass.getuser()
print(isinstance(user, str))      # True
print(len(user) > 0)              # True

# GetPassWarning is a class
print(isinstance(getpass.GetPassWarning, type))   # True

# GetPassWarning can be instantiated
w = getpass.GetPassWarning('test warning')
print(isinstance(w, getpass.GetPassWarning))   # True

# module has the expected attributes
print(hasattr(getpass, 'getpass'))         # True
print(hasattr(getpass, 'getuser'))         # True
print(hasattr(getpass, 'GetPassWarning'))  # True
