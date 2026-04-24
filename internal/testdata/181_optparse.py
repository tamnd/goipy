"""Tests for optparse module."""
import optparse

# --- Basic store option ---
p = optparse.OptionParser()
p.add_option('-f', '--file', dest='filename', default='out.txt')
opts, args = p.parse_args(['-f', 'input.txt'])
print(opts.filename == 'input.txt')   # True
print(args == [])                     # True

# --- default used when option absent ---
opts2, _ = p.parse_args([])
print(opts2.filename == 'out.txt')   # True

# --- long option with = ---
opts3, _ = p.parse_args(['--file=data.csv'])
print(opts3.filename == 'data.csv')   # True

# --- store_true / store_false ---
p2 = optparse.OptionParser()
p2.add_option('-v', '--verbose', action='store_true', dest='verbose', default=False)
p2.add_option('-q', '--quiet', action='store_false', dest='verbose')
opts4, _ = p2.parse_args(['-v'])
print(opts4.verbose == True)    # True
opts5, _ = p2.parse_args(['-q'])
print(opts5.verbose == False)   # True
opts6, _ = p2.parse_args([])
print(opts6.verbose == False)   # True

# --- store_const ---
p3 = optparse.OptionParser()
p3.add_option('--mode', action='store_const', const='fast', dest='mode', default='normal')
opts7, _ = p3.parse_args(['--mode'])
print(opts7.mode == 'fast')     # True
opts8, _ = p3.parse_args([])
print(opts8.mode == 'normal')   # True

# --- type conversion: int ---
p4 = optparse.OptionParser()
p4.add_option('-n', '--num', type='int', dest='num', default=0)
opts9, _ = p4.parse_args(['-n', '42'])
print(opts9.num == 42)    # True

# --- type conversion: float ---
p5 = optparse.OptionParser()
p5.add_option('--ratio', type='float', dest='ratio', default=1.0)
opts10, _ = p5.parse_args(['--ratio', '0.5'])
print(opts10.ratio == 0.5)   # True

# --- append action ---
p6 = optparse.OptionParser()
p6.add_option('--item', action='append', dest='items', default=[])
opts11, _ = p6.parse_args(['--item', 'a', '--item', 'b'])
print(opts11.items == ['a', 'b'])   # True

# --- count action ---
p7 = optparse.OptionParser()
p7.add_option('-v', action='count', dest='verbosity', default=0)
opts12, _ = p7.parse_args(['-v', '-v', '-v'])
print(opts12.verbosity == 3)   # True

# --- positional args returned in second element ---
p8 = optparse.OptionParser()
p8.add_option('--out', default='x')
opts13, args13 = p8.parse_args(['--out', 'y', 'pos1', 'pos2'])
print(opts13.out == 'y')              # True
print(args13 == ['pos1', 'pos2'])     # True

# --- set_defaults ---
p9 = optparse.OptionParser()
p9.add_option('--foo', default='bar')
p9.set_defaults(foo='baz', extra=99)
opts14, _ = p9.parse_args([])
print(opts14.foo == 'baz')   # True
print(opts14.extra == 99)    # True

# --- has_option ---
p10 = optparse.OptionParser()
p10.add_option('-x', '--xopt')
print(p10.has_option('-x'))       # True
print(p10.has_option('--xopt'))   # True
print(p10.has_option('--nope'))   # False

# --- get_option ---
p11 = optparse.OptionParser()
p11.add_option('-y', '--yopt', dest='y', default='Y')
opt_obj = p11.get_option('--yopt')
print(opt_obj is not None)   # True

# --- remove_option ---
p12 = optparse.OptionParser()
p12.add_option('--rm')
p12.remove_option('--rm')
print(p12.has_option('--rm') == False)   # True

# --- OptionGroup ---
p13 = optparse.OptionParser()
grp = p13.add_option_group('Advanced')
grp.add_option('--adv', default='adv_val')
opts15, _ = p13.parse_args([])
print(opts15.adv == 'adv_val')   # True

# --- Values.ensure_value ---
vals = optparse.Values({'x': None})
result = vals.ensure_value('x', 'default_x')
print(result == 'default_x')    # True
print(vals.x == 'default_x')    # True

# --- format_help returns str ---
p14 = optparse.OptionParser(prog='myprog', description='test')
print(isinstance(p14.format_help(), str))    # True
print(isinstance(p14.format_usage(), str))   # True

# --- make_option ---
opt = optparse.make_option('-z', '--zoo', dest='zoo', default='animals')
print(isinstance(opt, optparse.Option))   # True

# --- constants ---
print(optparse.SUPPRESS_HELP == 'SUPPRESS HELP')   # True

# --- exception classes exist ---
print(issubclass(optparse.OptionError, Exception))        # True
print(issubclass(optparse.OptionValueError, Exception))   # True
print(issubclass(optparse.BadOptionError, Exception))     # True

# --- dest auto-derived from long option ---
p15 = optparse.OptionParser()
p15.add_option('--my-opt', default='z')
opts16, _ = p15.parse_args(['--my-opt', 'w'])
print(opts16.my_opt == 'w')   # True

# --- -- stops option processing ---
p16 = optparse.OptionParser()
p16.add_option('--flag', action='store_true', default=False)
opts17, args17 = p16.parse_args(['--', '--flag'])
print(opts17.flag == False)          # True
print(args17 == ['--flag'])          # True

# --- nargs=2 for option ---
p17 = optparse.OptionParser()
p17.add_option('--point', nargs=2, type='int', dest='point')
opts18, _ = p17.parse_args(['--point', '3', '4'])
print(list(opts18.point) == [3, 4])   # True

# --- get_default_values ---
p18 = optparse.OptionParser()
p18.add_option('--val', default=7)
dv = p18.get_default_values()
print(dv.val == 7)   # True
