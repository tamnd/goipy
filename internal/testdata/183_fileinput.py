"""Tests for fileinput module."""
import fileinput
import tempfile
import os

# --- FileInput over explicit files ---
f1 = tempfile.mktemp(suffix='.txt')
f2 = tempfile.mktemp(suffix='.txt')
with open(f1, 'w') as fh:
    fh.write('line1\nline2\n')
with open(f2, 'w') as fh:
    fh.write('line3\nline4\n')

lines = []
with fileinput.FileInput(files=[f1, f2]) as fi:
    for line in fi:
        lines.append(line.rstrip('\n'))
print(lines == ['line1', 'line2', 'line3', 'line4'])   # True

os.remove(f1)
os.remove(f2)

# --- lineno() and filelineno() ---
f3 = tempfile.mktemp(suffix='.txt')
f4 = tempfile.mktemp(suffix='.txt')
with open(f3, 'w') as fh:
    fh.write('a\nb\n')
with open(f4, 'w') as fh:
    fh.write('c\n')

linenos = []
filelinenos = []
filenames = []
with fileinput.FileInput(files=[f3, f4]) as fi:
    for line in fi:
        linenos.append(fi.lineno())
        filelinenos.append(fi.filelineno())
        filenames.append(fi.filename())

print(linenos == [1, 2, 3])            # True
print(filelinenos == [1, 2, 1])        # True
print(filenames[0] == f3)              # True
print(filenames[2] == f4)              # True

os.remove(f3)
os.remove(f4)

# --- isfirstline() ---
f5 = tempfile.mktemp(suffix='.txt')
with open(f5, 'w') as fh:
    fh.write('x\ny\n')
first_flags = []
with fileinput.FileInput(files=[f5]) as fi:
    for line in fi:
        first_flags.append(fi.isfirstline())
print(first_flags == [True, False])   # True
os.remove(f5)

# --- isstdin() returns False for file input ---
f6 = tempfile.mktemp(suffix='.txt')
with open(f6, 'w') as fh:
    fh.write('z\n')
with fileinput.FileInput(files=[f6]) as fi:
    for line in fi:
        print(fi.isstdin() == False)   # True
os.remove(f6)

# --- nextfile() skips remaining lines in current file ---
f7 = tempfile.mktemp(suffix='.txt')
f8 = tempfile.mktemp(suffix='.txt')
with open(f7, 'w') as fh:
    fh.write('skip1\nskip2\nskip3\n')
with open(f8, 'w') as fh:
    fh.write('keep\n')

kept = []
with fileinput.FileInput(files=[f7, f8]) as fi:
    for line in fi:
        if fi.filename() == f7 and fi.filelineno() == 1:
            fi.nextfile()
        else:
            kept.append(line.rstrip('\n'))
print(kept == ['keep'])   # True

os.remove(f7)
os.remove(f8)

# --- fileinput.input() module-level function ---
f9 = tempfile.mktemp(suffix='.txt')
with open(f9, 'w') as fh:
    fh.write('mod1\nmod2\n')

fi2 = fileinput.input(files=[f9])
mlines = [l.rstrip('\n') for l in fi2]
fi2.close()
print(mlines == ['mod1', 'mod2'])   # True
os.remove(f9)

# --- module-level filename()/lineno() after input() ---
f10 = tempfile.mktemp(suffix='.txt')
with open(f10, 'w') as fh:
    fh.write('p\nq\n')

fi3 = fileinput.input(files=[f10])
collected = []
for line in fi3:
    collected.append((fileinput.filename(), fileinput.lineno(), fileinput.filelineno()))
fi3.close()
print(collected[0][0] == f10)   # True
print(collected[0][1] == 1)     # True
print(collected[1][1] == 2)     # True
os.remove(f10)

# --- FileInput with single filename string ---
f11 = tempfile.mktemp(suffix='.txt')
with open(f11, 'w') as fh:
    fh.write('one\ntwo\n')
with fileinput.FileInput(files=f11) as fi:
    result = list(fi)
print(len(result) == 2)   # True
print(result[0].rstrip('\n') == 'one')   # True
os.remove(f11)

# --- hook_encoded exists and is callable ---
hook = fileinput.hook_encoded('utf-8')
print(callable(hook))   # True

# --- hook_compressed exists and is callable ---
print(callable(fileinput.hook_compressed))   # True

# --- FileInput fileno() ---
f12 = tempfile.mktemp(suffix='.txt')
with open(f12, 'w') as fh:
    fh.write('test\n')
with fileinput.FileInput(files=[f12]) as fi:
    next(fi)
    print(fi.fileno() >= 0)   # True (file is open, has valid fd)
os.remove(f12)
