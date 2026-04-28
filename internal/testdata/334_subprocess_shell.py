"""v0.0.334 subprocess shell handling — shell=True with list args, executable= kwarg."""
import subprocess
from subprocess import run, Popen, PIPE


# ── 1. shell=True with list: argv[0] is the script, argv[1:] are positional args ($1, $2, …)
r = run(
    ['echo $0 / $1 / $2', 'first', 'second'],
    shell=True, capture_output=True, text=True,
)
assert r.returncode == 0, f"rc={r.returncode}"
# $0 is "first" (CPython passes argv[1] as $0 to the shell; with /bin/sh
# under -c, the first positional after the script string fills $0).
assert r.stdout == "first / second /\n", f"out={r.stdout!r}"
print("shell list ok")


# ── 2. shell=True with str: classic case still works
r = run("echo classic_shell", shell=True, capture_output=True, text=True)
assert r.returncode == 0
assert "classic_shell" in r.stdout
print("shell str ok")


# ── 3. shell=True + pipe still works (regression-check existing behaviour)
r = run("echo piped | cat", shell=True, capture_output=True, text=True)
assert r.returncode == 0
assert "piped" in r.stdout
print("shell pipe ok")


# ── 4. executable= kwarg in shell mode: explicit shell binary works
r = run("echo exec_shell", shell=True, executable="/bin/sh",
        capture_output=True, text=True)
assert r.returncode == 0
assert "exec_shell" in r.stdout
print("shell executable ok")


# ── 5. executable= kwarg in non-shell mode: program path replaces argv[0]
# We launch /bin/echo with argv = ["arg0", "arg1"]. echo treats argv[0] as
# its program name and prints the rest.
r = run(["arg0", "arg1"], executable="/bin/echo",
        capture_output=True, text=True)
assert r.returncode == 0
assert r.stdout.strip() == "arg1", f"out={r.stdout!r}"
print("nonshell executable ok")


# ── 6. getoutput / getstatusoutput still funnel through the right shell
out = subprocess.getoutput("echo getout_after")
assert "getout_after" in out, f"out={out!r}"
rc, out = subprocess.getstatusoutput("echo statout_after; false")
assert rc != 0, f"rc={rc}"
assert "statout_after" in out, f"out={out!r}"
print("getoutput ok")


# ── 7. shell=True positional args with quoted output
r = run(
    ['printf "%s\\n" "$@"', 'sh-name', 'a', 'b'],
    shell=True, capture_output=True, text=True,
)
# "$@" expands to $1 $2 ...; the script's $0 is 'sh-name'. So we expect
# 'a\nb\n'.
assert r.returncode == 0
assert r.stdout == "a\nb\n", f"out={r.stdout!r}"
print("shell positional ok")


print("ok")
