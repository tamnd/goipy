"""Tests for argparse module."""
import argparse

# --- Basic positional argument ---
p = argparse.ArgumentParser(prog='test')
p.add_argument('name')
ns = p.parse_args(['Alice'])
print(ns.name == 'Alice')   # True

# --- Optional argument (long) ---
p2 = argparse.ArgumentParser()
p2.add_argument('--count', type=int, default=0)
ns2 = p2.parse_args(['--count', '5'])
print(ns2.count == 5)   # True

# default used when not supplied
ns2b = p2.parse_args([])
print(ns2b.count == 0)  # True

# --- Optional argument (short) ---
p3 = argparse.ArgumentParser()
p3.add_argument('-v', '--verbose', action='store_true', default=False)
ns3 = p3.parse_args(['-v'])
print(ns3.verbose == True)   # True
ns3b = p3.parse_args([])
print(ns3b.verbose == False)  # True

# --- store_false ---
p4 = argparse.ArgumentParser()
p4.add_argument('--no-flag', dest='flag', action='store_false', default=True)
ns4 = p4.parse_args(['--no-flag'])
print(ns4.flag == False)   # True

# --- default without flag ---
ns4b = p4.parse_args([])
print(ns4b.flag == True)   # True

# --- store_const ---
p5 = argparse.ArgumentParser()
p5.add_argument('--mode', action='store_const', const='fast', default='slow')
ns5 = p5.parse_args(['--mode'])
print(ns5.mode == 'fast')   # True

# --- nargs='?' ---
p6 = argparse.ArgumentParser()
p6.add_argument('--out', nargs='?', const='stdout', default='file.txt')
ns6a = p6.parse_args(['--out', 'result.txt'])
print(ns6a.out == 'result.txt')   # True
ns6b = p6.parse_args(['--out'])
print(ns6b.out == 'stdout')       # True
ns6c = p6.parse_args([])
print(ns6c.out == 'file.txt')     # True

# --- nargs='*' ---
p7 = argparse.ArgumentParser()
p7.add_argument('items', nargs='*')
ns7 = p7.parse_args(['a', 'b', 'c'])
print(ns7.items == ['a', 'b', 'c'])   # True
ns7b = p7.parse_args([])
print(ns7b.items == [])               # True

# --- nargs='+' ---
p8 = argparse.ArgumentParser()
p8.add_argument('files', nargs='+')
ns8 = p8.parse_args(['x', 'y'])
print(ns8.files == ['x', 'y'])   # True

# --- nargs=N (integer) ---
p9 = argparse.ArgumentParser()
p9.add_argument('point', nargs=2, type=int)
ns9 = p9.parse_args(['3', '4'])
print(ns9.point == [3, 4])   # True

# --- append action ---
p10 = argparse.ArgumentParser()
p10.add_argument('--item', action='append')
ns10 = p10.parse_args(['--item', 'a', '--item', 'b'])
print(ns10.item == ['a', 'b'])   # True

# --- count action ---
p11 = argparse.ArgumentParser()
p11.add_argument('-v', action='count', default=0)
ns11 = p11.parse_args(['-v', '-v', '-v'])
print(ns11.v == 3)   # True

# --- type conversion ---
p12 = argparse.ArgumentParser()
p12.add_argument('x', type=float)
ns12 = p12.parse_args(['3.14'])
print(ns12.x == 3.14)   # True

# --- set_defaults ---
p13 = argparse.ArgumentParser()
p13.add_argument('--foo', default='bar')
p13.set_defaults(foo='baz', extra=42)
ns13 = p13.parse_args([])
print(ns13.foo == 'baz')    # True
print(ns13.extra == 42)     # True

# --- get_default ---
p14 = argparse.ArgumentParser()
p14.add_argument('--val', default=99)
print(p14.get_default('val') == 99)   # True

# --- Namespace ---
ns_obj = argparse.Namespace(x=1, y=2)
print(ns_obj.x == 1)          # True
print(ns_obj.y == 2)          # True
print('x' in ns_obj)          # True
print('z' not in ns_obj)      # True

# --- parse_known_args ---
p15 = argparse.ArgumentParser()
p15.add_argument('--known', default='k')
ns15, extra15 = p15.parse_known_args(['--known', 'val', '--unknown', 'x'])
print(ns15.known == 'val')         # True
print('--unknown' in extra15)      # True

# --- add_argument_group ---
p16 = argparse.ArgumentParser()
grp = p16.add_argument_group('mygroup')
grp.add_argument('--garg', default='gval')
ns16 = p16.parse_args([])
print(ns16.garg == 'gval')   # True

# --- mutually_exclusive_group ---
p17 = argparse.ArgumentParser()
meg = p17.add_mutually_exclusive_group()
meg.add_argument('--foo17', action='store_true')
meg.add_argument('--bar17', action='store_true')
ns17 = p17.parse_args(['--foo17'])
print(ns17.foo17 == True)    # True
print(ns17.bar17 == False)   # True

# --- formatter classes exist ---
print(hasattr(argparse, 'HelpFormatter'))                    # True
print(hasattr(argparse, 'RawDescriptionHelpFormatter'))      # True
print(hasattr(argparse, 'RawTextHelpFormatter'))             # True
print(hasattr(argparse, 'ArgumentDefaultsHelpFormatter'))    # True
print(hasattr(argparse, 'MetavarTypeHelpFormatter'))         # True

# --- constants ---
print(argparse.SUPPRESS == '==SUPPRESS==')   # True (CPython value)
print(argparse.REMAINDER == '...')           # True (CPython internal)

# --- format_usage / format_help return strings ---
p18 = argparse.ArgumentParser(prog='myprog', description='A test')
p18.add_argument('--opt', help='an option')
print(isinstance(p18.format_usage(), str))   # True
print(isinstance(p18.format_help(), str))    # True

# --- --long=value syntax ---
p19 = argparse.ArgumentParser()
p19.add_argument('--name', default='')
ns19 = p19.parse_args(['--name=hello'])
print(ns19.name == 'hello')   # True

# --- dest with dashes converted to underscores ---
p20 = argparse.ArgumentParser()
p20.add_argument('--my-arg', default='x')
ns20 = p20.parse_args(['--my-arg', 'y'])
print(ns20.my_arg == 'y')   # True

# --- FileType exists ---
print(callable(argparse.FileType))   # True

# --- ArgumentError, ArgumentTypeError exist ---
print(issubclass(argparse.ArgumentError, Exception))      # True
print(issubclass(argparse.ArgumentTypeError, Exception))  # True
