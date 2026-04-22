import string

print(string.ascii_lowercase)
print(string.ascii_uppercase)
print(string.ascii_letters)
print(string.digits)
print(string.hexdigits)
print(string.octdigits)
print(string.punctuation)
print(string.whitespace == ' \t\n\r\x0b\x0c')
print(string.printable)

# lengths
print(len(string.ascii_lowercase))   # 26
print(len(string.ascii_uppercase))   # 26
print(len(string.ascii_letters))     # 52
print(len(string.digits))            # 10
print(len(string.hexdigits))         # 22
print(len(string.octdigits))         # 8

# membership
print('a' in string.ascii_lowercase)   # True
print('Z' in string.ascii_uppercase)   # True
print('5' in string.digits)            # True
print('f' in string.hexdigits)         # True
print('8' in string.octdigits)         # False
print('!' in string.punctuation)       # True
print(' ' in string.whitespace)        # True
print('\t' in string.whitespace)       # True

# capwords
print(string.capwords('hello world'))           # Hello World
print(string.capwords('  hello   world  '))     # Hello World
print(string.capwords('hello-world', '-'))      # Hello-World
print(string.capwords('THE QUICK BROWN FOX'))   # The Quick Brown Fox
print(string.capwords('one,two,three', ','))    # One,Two,Three
