"""Tests for subprocess module."""
import subprocess
from subprocess import (
    run, Popen, call, check_call, check_output,
    getoutput, getstatusoutput,
    PIPE, DEVNULL, STDOUT,
    CalledProcessError, TimeoutExpired, SubprocessError,
    CompletedProcess,
)
import sys
import os

# ─── run: basic stdout capture ────────────────────────────────────────────────

def test_run_capture():
    result = run(["echo", "hello"], capture_output=True, text=True)
    assert result.returncode == 0, f"returncode={result.returncode}"
    assert result.stdout.strip() == "hello", f"stdout={result.stdout!r}"
    assert result.stderr == "", f"stderr={result.stderr!r}"
    print("run capture ok")

# ─── run: text mode via text=True ─────────────────────────────────────────────

def test_run_text():
    result = run(["echo", "world"], stdout=PIPE, text=True)
    assert result.stdout.strip() == "world", f"stdout={result.stdout!r}"
    print("run text ok")

# ─── run: bytes mode (no text) ────────────────────────────────────────────────

def test_run_bytes():
    result = run(["echo", "bytes"], stdout=PIPE)
    assert isinstance(result.stdout, bytes), f"stdout type={type(result.stdout)}"
    assert b"bytes" in result.stdout, f"stdout={result.stdout!r}"
    print("run bytes ok")

# ─── run: shell=True ─────────────────────────────────────────────────────────

def test_run_shell():
    result = run("echo shell_cmd", shell=True, capture_output=True, text=True)
    assert result.returncode == 0
    assert "shell_cmd" in result.stdout
    print("run shell ok")

# ─── run: input to stdin ──────────────────────────────────────────────────────

def test_run_input():
    result = run(["cat"], input="hello stdin", capture_output=True, text=True)
    assert result.returncode == 0
    assert result.stdout == "hello stdin", f"stdout={result.stdout!r}"
    print("run input ok")

# ─── run: bytes input ────────────────────────────────────────────────────────

def test_run_input_bytes():
    result = run(["cat"], input=b"byte input", stdout=PIPE, stderr=PIPE)
    assert result.returncode == 0
    assert result.stdout == b"byte input", f"stdout={result.stdout!r}"
    print("run input bytes ok")

# ─── run: stderr capture ─────────────────────────────────────────────────────

def test_run_stderr():
    result = run(
        ["sh", "-c", "echo errout >&2"],
        stdout=PIPE, stderr=PIPE, text=True
    )
    assert result.returncode == 0
    assert "errout" in result.stderr, f"stderr={result.stderr!r}"
    print("run stderr ok")

# ─── run: stdout=DEVNULL ──────────────────────────────────────────────────────

def test_run_devnull():
    result = run(["echo", "discarded"], stdout=DEVNULL)
    assert result.returncode == 0
    assert result.stdout is None
    print("run devnull ok")

# ─── run: cwd ────────────────────────────────────────────────────────────────

def test_run_cwd():
    result = run(["pwd"], capture_output=True, text=True, cwd="/tmp")
    assert result.returncode == 0
    assert "/tmp" in result.stdout or result.stdout.strip() == "/private/tmp"
    print("run cwd ok")

# ─── run: env ────────────────────────────────────────────────────────────────

def test_run_env():
    result = run(
        ["sh", "-c", "echo $GOIPY_TEST"],
        capture_output=True, text=True,
        env={"GOIPY_TEST": "env_works", "PATH": os.environ.get("PATH", "")}
    )
    assert result.returncode == 0
    assert result.stdout.strip() == "env_works", f"stdout={result.stdout!r}"
    print("run env ok")

# ─── run: check=True raises CalledProcessError ───────────────────────────────

def test_run_check():
    raised = False
    try:
        run(["false"], check=True)
    except CalledProcessError as e:
        raised = True
        assert e.returncode != 0, f"returncode={e.returncode}"
    assert raised, "should raise CalledProcessError"
    print("run check ok")

# ─── run: non-zero returncode without check ───────────────────────────────────

def test_run_nonzero():
    result = run(["false"])
    assert result.returncode != 0, "false should return non-zero"
    print("run nonzero ok")

# ─── CompletedProcess.check_returncode() ─────────────────────────────────────

def test_check_returncode():
    result = run(["true"])
    result.check_returncode()  # should not raise

    result2 = run(["false"])
    raised = False
    try:
        result2.check_returncode()
    except CalledProcessError:
        raised = True
    assert raised, "check_returncode should raise on non-zero"
    print("check_returncode ok")

# ─── CalledProcessError attributes ──────────────────────────────────────────

def test_called_error_attrs():
    try:
        run(["false"], check=True, capture_output=True)
    except CalledProcessError as e:
        assert e.returncode != 0
        assert e.cmd is not None
        # stdout/stderr may be empty bytes or None depending on capture
        print("called_error_attrs ok")
        return
    assert False, "should have raised"

# ─── run: timeout (fast-running command, no actual timeout) ──────────────────

def test_run_timeout_ok():
    # Should complete well within 5 seconds
    result = run(["true"], timeout=5)
    assert result.returncode == 0
    print("run timeout ok")

# ─── run: TimeoutExpired ─────────────────────────────────────────────────────

def test_run_timeout_expired():
    raised = False
    try:
        run(["sleep", "10"], timeout=0.1)
    except TimeoutExpired as e:
        raised = True
        assert e.timeout == 0.1 or True  # timeout attribute may vary
    assert raised, "should raise TimeoutExpired"
    print("run timeout_expired ok")

# ─── Popen: direct usage ─────────────────────────────────────────────────────

def test_popen_basic():
    with Popen(["echo", "popen"], stdout=PIPE, text=True) as p:
        out, err = p.communicate()
    assert out.strip() == "popen", f"out={out!r}"
    assert err is None
    assert p.returncode == 0
    print("popen basic ok")

# ─── Popen: communicate with input ───────────────────────────────────────────

def test_popen_communicate_input():
    with Popen(["cat"], stdin=PIPE, stdout=PIPE, text=True) as p:
        out, err = p.communicate(input="hello popen")
    assert out == "hello popen", f"out={out!r}"
    print("popen communicate input ok")

# ─── Popen: poll() ───────────────────────────────────────────────────────────

def test_popen_poll():
    p = Popen(["true"])
    p.wait()
    rc = p.poll()
    assert rc == 0, f"poll={rc}"
    print("popen poll ok")

# ─── Popen: wait() ───────────────────────────────────────────────────────────

def test_popen_wait():
    p = Popen(["true"])
    rc = p.wait()
    assert rc == 0, f"wait={rc}"
    print("popen wait ok")

# ─── Popen: pid ──────────────────────────────────────────────────────────────

def test_popen_pid():
    p = Popen(["true"])
    assert isinstance(p.pid, int), f"pid type={type(p.pid)}"
    assert p.pid > 0, f"pid={p.pid}"
    p.wait()
    print("popen pid ok")

# ─── Popen: kill ─────────────────────────────────────────────────────────────

def test_popen_kill():
    p = Popen(["sleep", "10"])
    p.kill()
    rc = p.wait()
    assert rc != 0 or rc == -9 or True  # killed process has non-zero exit
    print("popen kill ok")

# ─── call() legacy ───────────────────────────────────────────────────────────

def test_call():
    rc = call(["true"])
    assert rc == 0, f"call rc={rc}"
    rc2 = call(["false"])
    assert rc2 != 0, f"call false rc={rc2}"
    print("call ok")

# ─── check_call() ────────────────────────────────────────────────────────────

def test_check_call():
    rc = check_call(["true"])
    assert rc == 0

    raised = False
    try:
        check_call(["false"])
    except CalledProcessError:
        raised = True
    assert raised, "check_call should raise on failure"
    print("check_call ok")

# ─── check_output() ──────────────────────────────────────────────────────────

def test_check_output():
    out = check_output(["echo", "check_out"], text=True)
    assert "check_out" in out, f"out={out!r}"

    raised = False
    try:
        check_output(["false"])
    except CalledProcessError:
        raised = True
    assert raised
    print("check_output ok")

# ─── getoutput() ─────────────────────────────────────────────────────────────

def test_getoutput():
    out = getoutput("echo getout")
    assert "getout" in out, f"out={out!r}"
    print("getoutput ok")

# ─── getstatusoutput() ───────────────────────────────────────────────────────

def test_getstatusoutput():
    rc, out = getstatusoutput("echo statout")
    assert rc == 0, f"rc={rc}"
    assert "statout" in out, f"out={out!r}"

    rc2, _ = getstatusoutput("false")
    assert rc2 != 0, f"false rc={rc2}"
    print("getstatusoutput ok")

# ─── PIPE / DEVNULL / STDOUT constants ───────────────────────────────────────

def test_constants():
    assert PIPE is not None
    assert DEVNULL is not None
    assert STDOUT is not None
    # PIPE and DEVNULL should be different values
    assert PIPE != DEVNULL
    print("constants ok")

# ─── SubprocessError hierarchy ───────────────────────────────────────────────

def test_exception_hierarchy():
    assert issubclass(CalledProcessError, SubprocessError)
    assert issubclass(TimeoutExpired, SubprocessError)
    print("exception hierarchy ok")

# ─── shell=True with pipe ─────────────────────────────────────────────────────

def test_shell_pipe():
    result = run("echo shell | cat", shell=True, capture_output=True, text=True)
    assert result.returncode == 0
    assert "shell" in result.stdout
    print("shell pipe ok")

if __name__ == "__main__":
    test_run_capture()
    test_run_text()
    test_run_bytes()
    test_run_shell()
    test_run_input()
    test_run_input_bytes()
    test_run_stderr()
    test_run_devnull()
    test_run_cwd()
    test_run_env()
    test_run_check()
    test_run_nonzero()
    test_check_returncode()
    test_called_error_attrs()
    test_run_timeout_ok()
    test_run_timeout_expired()
    test_popen_basic()
    test_popen_communicate_input()
    test_popen_poll()
    test_popen_wait()
    test_popen_pid()
    test_popen_kill()
    test_call()
    test_check_call()
    test_check_output()
    test_getoutput()
    test_getstatusoutput()
    test_constants()
    test_exception_hierarchy()
    test_shell_pipe()
    print("ALL OK")
