from string import Template

# --- Basic substitution ---
t = Template('$who likes $what')
print(t.substitute(who='tim', what='kung pao'))   # tim likes kung pao

# --- Braced substitution ---
t2 = Template('${name}!')
print(t2.substitute(name='hi'))                   # hi!

# --- Dollar escape ---
t3 = Template('Price: $$100')
print(t3.substitute())                            # Price: $100

# --- Mapping dict ---
d = {'animal': 'dog', 'sound': 'woof'}
t4 = Template('The $animal says $sound')
print(t4.substitute(d))                           # The dog says woof

# --- Mixed mapping + kwargs ---
t5 = Template('$first $last')
print(t5.substitute({'first': 'John'}, last='Doe'))  # John Doe

# --- safe_substitute: missing key left unchanged ---
t6 = Template('$who likes $what')
print(t6.safe_substitute(who='tim'))              # tim likes $what

t7 = Template('Hello $name')
print(t7.safe_substitute())                       # Hello $name

# --- safe_substitute with braced ---
t8 = Template('${greeting}, ${name}!')
print(t8.safe_substitute(greeting='Hello'))       # Hello, ${name}!

# --- .template attribute ---
t9 = Template('$x + $y')
print(t9.template)                                # $x + $y

# --- is_valid() ---
print(Template('$name').is_valid())               # True
print(Template('${name}').is_valid())             # True
print(Template('$$').is_valid())                  # True
print(Template('$$$').is_valid())                 # False (trailing $)

# --- get_identifiers() ---
t10 = Template('$first and $second and $first again')
print(t10.get_identifiers())                      # ['first', 'second']

t11 = Template('${x} + ${y} = ${z}')
print(t11.get_identifiers())                      # ['x', 'y', 'z']

# --- KeyError from substitute with missing key ---
try:
    Template('$name').substitute()
except KeyError:
    print('KeyError raised')                      # KeyError raised

# --- Mixed: dollar-sign before non-identifier (safe_substitute leaves it) ---
t12 = Template('cost: $$ each')
print(t12.safe_substitute())                      # cost: $ each
