"""Tests for the cmd module."""
import cmd
import io

# --- module has Cmd class ---
print(hasattr(cmd, 'Cmd'))                               # True

class Shell(cmd.Cmd):
    def do_hello(self, arg):
        """Say hello."""
        self.stdout.write('hello ' + arg + '\n')

    def do_add(self, arg):
        parts = arg.split()
        self.stdout.write(str(sum(int(x) for x in parts)) + '\n')

    def do_quit(self, arg):
        """Quit."""
        return True

buf = io.StringIO()
sh = Shell(stdout=buf)

# --- isinstance ---
print(isinstance(sh, cmd.Cmd))                          # True

# --- prompt default ---
print(sh.prompt == '(Cmd) ')                            # True

# --- parseline: empty ---
print(sh.parseline('')[0] is None)                      # True

# --- parseline: normal command ---
print(sh.parseline('hello world')[0])                   # hello
print(sh.parseline('hello world')[1])                   # world

# --- parseline: ? maps to help ---
print(sh.parseline('?')[0])                             # help

# --- parseline: ! without do_shell -> None ---
print(sh.parseline('!ls')[0] is None)                   # True

# --- parseline: ! with do_shell -> shell ---
class ShellWithBang(cmd.Cmd):
    def do_shell(self, arg):
        self.stdout.write('ran: ' + arg + '\n')
    def do_quit(self, arg):
        return True

sw = ShellWithBang(stdout=io.StringIO())
print(sw.parseline('!ls')[0])                           # shell

# --- onecmd dispatches to do_hello ---
buf2 = io.StringIO()
sh2 = Shell(stdout=buf2)
sh2.onecmd('hello world')
print(buf2.getvalue().strip())                          # hello world

# --- onecmd returns stop flag from do_quit ---
buf3 = io.StringIO()
sh3 = Shell(stdout=buf3)
print(sh3.onecmd('quit') == True)                       # True

# --- onecmd: unknown command writes *** Unknown syntax ---
buf4 = io.StringIO()
sh4 = Shell(stdout=buf4)
sh4.onecmd('nosuchcmd foo')
print(len(buf4.getvalue()) > 0)                         # True

# --- onecmd: do_add computes sum ---
buf_a = io.StringIO()
sh_a = Shell(stdout=buf_a)
sh_a.onecmd('add 1 2 3')
print(buf_a.getvalue().strip())                         # 6

# --- cmdloop with preloaded cmdqueue ---
buf5 = io.StringIO()
sh5 = Shell(stdout=buf5)
sh5.cmdqueue = ['hello cmdloop', 'quit']
sh5.cmdloop(intro='')
print('hello cmdloop' in buf5.getvalue())               # True

# --- preloop / postloop called ---
class LoopShell(cmd.Cmd):
    def __init__(self, **kw):
        super().__init__(**kw)
        self.loops = []

    def preloop(self):
        self.loops.append('pre')

    def postloop(self):
        self.loops.append('post')

    def do_quit(self, arg):
        return True

ls = LoopShell(stdout=io.StringIO())
ls.cmdqueue = ['quit']
ls.cmdloop(intro='')
print(ls.loops[0] == 'pre')                             # True
print(ls.loops[1] == 'post')                            # True

# --- precmd / postcmd hooks (called by cmdloop, not onecmd) ---
class HookedShell(cmd.Cmd):
    def __init__(self, **kw):
        super().__init__(**kw)
        self.pre = []
        self.post = []

    def do_go(self, arg):
        pass

    def do_stop(self, arg):
        return True

    def precmd(self, line):
        self.pre.append(line)
        return line

    def postcmd(self, stop, line):
        self.post.append(line)
        return stop

hs = HookedShell(stdout=io.StringIO())
hs.cmdqueue = ['go', 'stop']
hs.cmdloop(intro='')
print(len(hs.pre) >= 1)                                 # True
print('go' in hs.pre)                                   # True

# --- do_help lists documented commands ---
buf_h = io.StringIO()
sh_h = Shell(stdout=buf_h)
sh_h.do_help('')
out = buf_h.getvalue()
print('hello' in out)                                   # True
print('quit' in out)                                    # True

# --- do_help with arg: prints docstring ---
buf_hh = io.StringIO()
sh_hh = Shell(stdout=buf_hh)
sh_hh.do_help('hello')
print(len(buf_hh.getvalue()) > 0)                       # True

# --- identchars membership ---
print('a' in sh.identchars)                             # True
print('_' in sh.identchars)                            # True
print('0' in sh.identchars)                            # True

# --- ruler ---
print(sh.ruler == '=')                                  # True

# --- columnize writes output ---
buf_c = io.StringIO()
sh_c = Shell(stdout=buf_c)
sh_c.columnize(['alpha', 'beta', 'gamma'])
print(len(buf_c.getvalue()) > 0)                        # True

# --- header attributes are strings ---
print(isinstance(sh.doc_header, str))                   # True
print(isinstance(sh.undoc_header, str))                 # True
print(isinstance(sh.misc_header, str))                  # True

# --- use_rawinput default ---
print(sh.use_rawinput == True)                          # True

# --- cmdloop prints intro ---
buf_i = io.StringIO()
sh_i = Shell(stdout=buf_i)
sh_i.cmdqueue = ['quit']
sh_i.cmdloop(intro='Welcome!')
print('Welcome!' in buf_i.getvalue())                   # True

# --- lastcmd initially empty ---
print(sh.lastcmd == '')                                 # True
