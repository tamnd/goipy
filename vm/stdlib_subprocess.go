package vm

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/tamnd/goipy/object"
)

// spPIPE / spDEVNULL / spSTDOUT mirror subprocess.PIPE etc.
const (
	spPIPE    = -1
	spDEVNULL = -2
	spSTDOUT  = -3
)

// spPopenState holds the mutable state of a Popen instance.
type spPopenState struct {
	cmd        *exec.Cmd
	argsObj    object.Object // original args for .args attribute
	textMode   bool

	// Per-pipe buffers set up before Start().
	stdinPipe  io.WriteCloser // non-nil when stdin=PIPE
	stdoutBuf  *bytes.Buffer  // non-nil when stdout=PIPE
	stderrBuf  *bytes.Buffer  // non-nil when stderr=PIPE (and not STDOUT)
	stderrIsStdout bool       // stderr=STDOUT

	mu         sync.Mutex
	started    bool
	done       bool
	returncode int
}

// buildSubprocess constructs the subprocess module.
func (i *Interp) buildSubprocess() *object.Module {
	m := &object.Module{Name: "subprocess", Dict: object.NewDict()}

	// ─── Constants ────────────────────────────────────────────────────────
	m.Dict.SetStr("PIPE",    object.NewInt(spPIPE))
	m.Dict.SetStr("DEVNULL", object.NewInt(spDEVNULL))
	m.Dict.SetStr("STDOUT",  object.NewInt(spSTDOUT))

	// ─── Exception classes ─────────────────────────────────────────────────
	subprocErr  := &object.Class{Name: "SubprocessError", Bases: []*object.Class{i.runtimeErr}, Dict: object.NewDict()}
	calledProcErr := &object.Class{Name: "CalledProcessError", Bases: []*object.Class{subprocErr}, Dict: object.NewDict()}
	timeoutExpired := &object.Class{Name: "TimeoutExpired", Bases: []*object.Class{subprocErr}, Dict: object.NewDict()}

	// CompletedProcess class (exposed for isinstance checks).
	completedProcCls := &object.Class{Name: "CompletedProcess", Dict: object.NewDict()}
	m.Dict.SetStr("CompletedProcess",   completedProcCls)
	m.Dict.SetStr("SubprocessError",    subprocErr)
	m.Dict.SetStr("CalledProcessError", calledProcErr)
	m.Dict.SetStr("TimeoutExpired",     timeoutExpired)

	// ─── Popen ────────────────────────────────────────────────────────────
	m.Dict.SetStr("Popen", &object.BuiltinFunc{Name: "Popen",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return i.makeSubprocPopen(a, kw, calledProcErr, timeoutExpired)
		}})

	// ─── run() ────────────────────────────────────────────────────────────
	m.Dict.SetStr("run", &object.BuiltinFunc{Name: "run",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return i.spRun(a, kw, completedProcCls, calledProcErr, timeoutExpired)
		}})

	// ─── call() ───────────────────────────────────────────────────────────
	m.Dict.SetStr("call", &object.BuiltinFunc{Name: "call",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			cp, err := i.spRun(a, kw, completedProcCls, calledProcErr, timeoutExpired)
			if err != nil {
				return nil, err
			}
			rc, _ := cp.(*object.Instance).Dict.GetStr("returncode")
			return rc, nil
		}})

	// ─── check_call() ─────────────────────────────────────────────────────
	m.Dict.SetStr("check_call", &object.BuiltinFunc{Name: "check_call",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if kw == nil {
				kw = object.NewDict()
			}
			kw.SetStr("check", object.True)
			_, err := i.spRun(a, kw, completedProcCls, calledProcErr, timeoutExpired)
			if err != nil {
				return nil, err
			}
			return object.NewInt(0), nil
		}})

	// ─── check_output() ───────────────────────────────────────────────────
	m.Dict.SetStr("check_output", &object.BuiltinFunc{Name: "check_output",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if kw == nil {
				kw = object.NewDict()
			}
			kw.SetStr("stdout", object.NewInt(spPIPE))
			kw.SetStr("check", object.True)
			cp, err := i.spRun(a, kw, completedProcCls, calledProcErr, timeoutExpired)
			if err != nil {
				return nil, err
			}
			if v, ok := cp.(*object.Instance).Dict.GetStr("stdout"); ok {
				return v, nil
			}
			return object.None, nil
		}})

	// ─── getoutput() ──────────────────────────────────────────────────────
	m.Dict.SetStr("getoutput", &object.BuiltinFunc{Name: "getoutput",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "getoutput() requires cmd argument")
			}
			cmd, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "getoutput() cmd must be str")
			}
			c := exec.Command("sh", "-c", cmd.V)
			c.Stderr = nil
			out, _ := c.Output()
			return &object.Str{V: strings.TrimRight(string(out), "\n")}, nil
		}})

	// ─── getstatusoutput() ────────────────────────────────────────────────
	m.Dict.SetStr("getstatusoutput", &object.BuiltinFunc{Name: "getstatusoutput",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "getstatusoutput() requires cmd argument")
			}
			cmd, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "getstatusoutput() cmd must be str")
			}
			var outBuf bytes.Buffer
			c := exec.Command("sh", "-c", cmd.V)
			c.Stdout = &outBuf
			c.Stderr = &outBuf
			runErr := c.Run()
			rc := 0
			if runErr != nil {
				if exitErr, ok2 := runErr.(*exec.ExitError); ok2 {
					rc = exitErr.ExitCode()
				} else {
					rc = 1
				}
			}
			output := strings.TrimRight(outBuf.String(), "\n")
			return &object.Tuple{V: []object.Object{
				object.NewInt(int64(rc)),
				&object.Str{V: output},
			}}, nil
		}})

	return m
}

// ─── Popen constructor ───────────────────────────────────────────────────────

func (i *Interp) makeSubprocPopen(
	a []object.Object, kw *object.Dict,
	calledProcErr, timeoutExpired *object.Class,
) (*object.Instance, error) {
	// Parse parameters.
	var (
		argsObj    object.Object
		stdinMode  = 0  // 0=inherit, spPIPE, spDEVNULL
		stdoutMode = 0
		stderrMode = 0
		shellMode  = false
		cwdStr     = ""
		textMode   = false
		envMap     *object.Dict
	)

	if len(a) > 0 {
		argsObj = a[0]
	}
	if kw != nil {
		if v, ok := kw.GetStr("args"); ok {
			argsObj = v
		}
		if v, ok := kw.GetStr("stdin"); ok {
			if n, ok2 := toInt64(v); ok2 {
				stdinMode = int(n)
			}
		}
		if v, ok := kw.GetStr("stdout"); ok {
			if n, ok2 := toInt64(v); ok2 {
				stdoutMode = int(n)
			}
		}
		if v, ok := kw.GetStr("stderr"); ok {
			if n, ok2 := toInt64(v); ok2 {
				stderrMode = int(n)
			}
		}
		if v, ok := kw.GetStr("capture_output"); ok {
			if b, ok2 := v.(*object.Bool); ok2 && b.V {
				stdoutMode = spPIPE
				stderrMode = spPIPE
			}
		}
		if v, ok := kw.GetStr("shell"); ok {
			if b, ok2 := v.(*object.Bool); ok2 {
				shellMode = b.V
			}
		}
		if v, ok := kw.GetStr("cwd"); ok {
			if s, ok2 := v.(*object.Str); ok2 {
				cwdStr = s.V
			}
		}
		if v, ok := kw.GetStr("text"); ok {
			if b, ok2 := v.(*object.Bool); ok2 {
				textMode = b.V
			}
		}
		if v, ok := kw.GetStr("universal_newlines"); ok {
			if b, ok2 := v.(*object.Bool); ok2 {
				textMode = textMode || b.V
			}
		}
		if v, ok := kw.GetStr("encoding"); ok {
			if _, ok2 := v.(*object.Str); ok2 {
				textMode = true // encoding implies text mode
			}
		}
		if v, ok := kw.GetStr("env"); ok {
			if d, ok2 := v.(*object.Dict); ok2 {
				envMap = d
			}
		}
	}
	if argsObj == nil {
		return nil, object.Errorf(i.typeErr, "Popen() requires args argument")
	}

	// Build argv.
	argv, err := spBuildArgv(argsObj, shellMode)
	if err != nil {
		return nil, object.Errorf(i.typeErr, "Popen() args: %v", err)
	}

	ps := &spPopenState{argsObj: argsObj, textMode: textMode}

	if shellMode {
		ps.cmd = exec.Command("sh", "-c", strings.Join(argv, " "))
	} else {
		if len(argv) == 0 {
			return nil, object.Errorf(i.typeErr, "Popen() args must not be empty")
		}
		ps.cmd = exec.Command(argv[0], argv[1:]...)
	}

	if cwdStr != "" {
		ps.cmd.Dir = cwdStr
	}
	if envMap != nil {
		ps.cmd.Env = spDictToEnv(envMap)
	}

	// Stdin.
	switch stdinMode {
	case spPIPE:
		pr, pw, perr := os.Pipe()
		if perr != nil {
			return nil, object.Errorf(i.runtimeErr, "Popen stdin pipe: %v", perr)
		}
		ps.cmd.Stdin = pr
		ps.stdinPipe = pw
		// We'll close pr after Start.
		defer func() { pr.Close() }()
	case spDEVNULL:
		ps.cmd.Stdin = io.NopCloser(strings.NewReader(""))
	}

	// Stdout.
	switch stdoutMode {
	case spPIPE:
		ps.stdoutBuf = &bytes.Buffer{}
		ps.cmd.Stdout = ps.stdoutBuf
	case spDEVNULL:
		ps.cmd.Stdout = io.Discard
	}

	// Stderr.
	switch stderrMode {
	case spPIPE:
		if ps.stdoutBuf == nil {
			ps.stderrBuf = &bytes.Buffer{}
			ps.cmd.Stderr = ps.stderrBuf
		} else {
			ps.stderrBuf = &bytes.Buffer{}
			ps.cmd.Stderr = ps.stderrBuf
		}
	case spDEVNULL:
		ps.cmd.Stderr = io.Discard
	case spSTDOUT:
		ps.stderrIsStdout = true
		if ps.stdoutBuf != nil {
			ps.cmd.Stderr = ps.stdoutBuf
		}
		// else: both inherit
	}

	if startErr := ps.cmd.Start(); startErr != nil {
		return nil, object.Errorf(i.osErr, "Popen: %v", startErr)
	}
	ps.started = true

	return i.spPopenInstance(ps, calledProcErr, timeoutExpired), nil
}

func (i *Interp) spPopenInstance(ps *spPopenState, calledProcErr, timeoutExpired *object.Class) *object.Instance {
	cls := &object.Class{Name: "Popen", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	// Set .args
	inst.Dict.SetStr("args", ps.argsObj)

	// Set .pid
	var pid int
	if ps.cmd.Process != nil {
		pid = ps.cmd.Process.Pid
	}
	inst.Dict.SetStr("pid", object.NewInt(int64(pid)))

	// returncode starts as None (not yet waited)
	inst.Dict.SetStr("returncode", object.None)

	// stdin/stdout/stderr stream attributes
	if ps.stdinPipe != nil {
		inst.Dict.SetStr("stdin", spMakeWriteStream(ps.stdinPipe))
	} else {
		inst.Dict.SetStr("stdin", object.None)
	}
	if ps.stdoutBuf != nil {
		// stdout is a lazy reader: data only available after wait/communicate
		inst.Dict.SetStr("stdout", spMakeLazyReader(ps, true))
	} else {
		inst.Dict.SetStr("stdout", object.None)
	}
	if ps.stderrBuf != nil {
		inst.Dict.SetStr("stderr", spMakeLazyReader(ps, false))
	} else {
		inst.Dict.SetStr("stderr", object.None)
	}

	// wait finalizes the process
	doWait := func() int {
		ps.mu.Lock()
		if ps.done {
			rc := ps.returncode
			ps.mu.Unlock()
			return rc
		}
		ps.mu.Unlock()

		waitErr := ps.cmd.Wait()
		rc := 0
		if waitErr != nil {
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				rc = exitErr.ExitCode()
			} else {
				rc = 1
			}
		}
		ps.mu.Lock()
		ps.done = true
		ps.returncode = rc
		ps.mu.Unlock()
		inst.Dict.SetStr("returncode", object.NewInt(int64(rc)))
		return rc
	}

	// poll()
	cls.Dict.SetStr("poll", &object.BuiltinFunc{Name: "poll",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ps.mu.Lock()
			done := ps.done
			rc := ps.returncode
			ps.mu.Unlock()
			if done {
				return object.NewInt(int64(rc)), nil
			}
			if ps.cmd.Process == nil {
				return object.None, nil
			}
			// Non-blocking: check via Process.Wait with WNOHANG via os.FindProcess trick
			// Simplest: try cmd.ProcessState
			if ps.cmd.ProcessState != nil {
				rc2 := doWait()
				return object.NewInt(int64(rc2)), nil
			}
			return object.None, nil
		}})

	// wait(timeout=None)
	cls.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			timeout := cfParseTimeout(a, kw, "timeout")
			if timeout < 0 {
				rc := doWait()
				return object.NewInt(int64(rc)), nil
			}
			done := make(chan int, 1)
			go func() { done <- doWait() }()
			select {
			case rc := <-done:
				return object.NewInt(int64(rc)), nil
			case <-time.After(timeout):
				return nil, object.Errorf(timeoutExpired, "Command timed out")
			}
		}})

	// communicate(input=None, timeout=None)
	cls.Dict.SetStr("communicate", &object.BuiltinFunc{Name: "communicate",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			var inputData []byte
			if len(a) > 0 {
				switch t := a[0].(type) {
				case *object.Bytes:
					inputData = t.V
				case *object.Str:
					inputData = []byte(t.V)
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("input"); ok {
					switch t := v.(type) {
					case *object.Bytes:
						inputData = t.V
					case *object.Str:
						inputData = []byte(t.V)
					}
				}
			}
			timeoutArgs := []object.Object(nil)
			if len(a) > 1 {
				timeoutArgs = a[1:]
			}
			timeout := cfParseTimeout(timeoutArgs, kw, "timeout")

			// Capture original timeout Python object for TimeoutExpired.
			var timeoutPyObj object.Object = object.None
			if kw != nil {
				if v, ok2 := kw.GetStr("timeout"); ok2 {
					timeoutPyObj = v
				}
			}
			if len(timeoutArgs) > 0 {
				timeoutPyObj = timeoutArgs[0]
			}

			// Write to stdin pipe if available.
			if ps.stdinPipe != nil && len(inputData) > 0 {
				ps.stdinPipe.Write(inputData) //nolint
			}
			if ps.stdinPipe != nil {
				ps.stdinPipe.Close()
				ps.stdinPipe = nil
			}

			makeTimeoutErr := func() error {
				exc := object.NewException(timeoutExpired, "Command timed out")
				exc.Dict = object.NewDict()
				exc.Dict.SetStr("cmd",     ps.argsObj)
				exc.Dict.SetStr("timeout", timeoutPyObj)
				exc.Dict.SetStr("output",  object.None)
				exc.Dict.SetStr("stdout",  object.None)
				exc.Dict.SetStr("stderr",  object.None)
				return exc
			}

			var rc int
			if timeout < 0 {
				rc = doWait()
			} else {
				done := make(chan int, 1)
				go func() { done <- doWait() }()
				select {
				case rc = <-done:
				case <-time.After(timeout):
					if ps.cmd.Process != nil {
						ps.cmd.Process.Kill() //nolint
					}
					return nil, makeTimeoutErr()
				}
			}
			_ = rc

			var stdoutVal, stderrVal object.Object = object.None, object.None
			if ps.stdoutBuf != nil {
				data := ps.stdoutBuf.Bytes()
				if ps.textMode {
					stdoutVal = &object.Str{V: string(data)}
				} else {
					stdoutVal = &object.Bytes{V: append([]byte(nil), data...)}
				}
			}
			if ps.stderrBuf != nil {
				data := ps.stderrBuf.Bytes()
				if ps.textMode {
					stderrVal = &object.Str{V: string(data)}
				} else {
					stderrVal = &object.Bytes{V: append([]byte(nil), data...)}
				}
			}
			return &object.Tuple{V: []object.Object{stdoutVal, stderrVal}}, nil
		}})

	// send_signal(sig)
	cls.Dict.SetStr("send_signal", &object.BuiltinFunc{Name: "send_signal",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 || ps.cmd.Process == nil {
				return object.None, nil
			}
			if n, ok := toInt64(a[0]); ok {
				ps.cmd.Process.Signal(syscall.Signal(n)) //nolint
			}
			return object.None, nil
		}})

	// terminate() → SIGTERM
	cls.Dict.SetStr("terminate", &object.BuiltinFunc{Name: "terminate",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if ps.cmd.Process != nil {
				ps.cmd.Process.Signal(syscall.SIGTERM) //nolint
			}
			return object.None, nil
		}})

	// kill() → SIGKILL
	cls.Dict.SetStr("kill", &object.BuiltinFunc{Name: "kill",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if ps.cmd.Process != nil {
				ps.cmd.Process.Kill() //nolint
			}
			return object.None, nil
		}})

	// Context manager.
	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})
	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			doWait()
			if ps.stdinPipe != nil {
				ps.stdinPipe.Close()
			}
			return object.False, nil
		}})

	return inst
}

// ─── run() ───────────────────────────────────────────────────────────────────

func (i *Interp) spRun(
	a []object.Object, kw *object.Dict,
	completedProcCls, calledProcErr, timeoutExpired *object.Class,
) (object.Object, error) {
	// Extract run()-specific params before delegating to Popen.
	var inputData object.Object
	checkFlag := false
	var timeoutVal object.Object

	var runKw *object.Dict
	if kw != nil {
		// Clone kw so we can modify it for Popen without side effects.
		ks, vs := kw.Items()
		runKw = object.NewDict()
		for idx, k := range ks {
			runKw.Set(k, vs[idx]) //nolint
		}
	} else {
		runKw = object.NewDict()
	}

	if v, ok := runKw.GetStr("input"); ok {
		inputData = v
		// Remove 'input' from kw so Popen doesn't see it;
		// also set stdin=PIPE if input is provided.
		runKw.SetStr("stdin", object.NewInt(spPIPE))
		// Remove 'input' key: Popen doesn't know it.
		// (Popen simply ignores unknown kwargs, but let's keep it clean.)
	}
	if v, ok := runKw.GetStr("check"); ok {
		if b, ok2 := v.(*object.Bool); ok2 {
			checkFlag = b.V
		}
	}
	if v, ok := runKw.GetStr("timeout"); ok {
		timeoutVal = v
	}

	popenInst, err := i.makeSubprocPopen(a, runKw, calledProcErr, timeoutExpired)
	if err != nil {
		return nil, err
	}

	// Get the spPopenState so we can call communicate.
	// We do this by calling communicate directly.
	commArgs := []object.Object{}
	commKw := object.NewDict()
	if inputData != nil {
		commKw.SetStr("input", inputData)
	}
	if timeoutVal != nil {
		commKw.SetStr("timeout", timeoutVal)
	}
	// Call communicate via the class method.
	commFn, ok := popenInst.Class.Dict.GetStr("communicate")
	if !ok {
		return nil, object.Errorf(i.runtimeErr, "Popen missing communicate")
	}
	commResult, commErr := commFn.(*object.BuiltinFunc).Call(nil,
		append([]object.Object{popenInst}, commArgs...), commKw)
	if commErr != nil {
		return nil, commErr
	}

	// Extract stdout, stderr from communicate result tuple.
	var stdoutVal, stderrVal object.Object = object.None, object.None
	if tup, ok2 := commResult.(*object.Tuple); ok2 && len(tup.V) == 2 {
		stdoutVal = tup.V[0]
		stderrVal = tup.V[1]
	}

	// Get returncode.
	rcObj, _ := popenInst.Dict.GetStr("returncode")
	rc := int64(0)
	if n, ok2 := rcObj.(*object.Int); ok2 && n.IsInt64() {
		rc = n.Int64()
	}

	// Build CalledProcessError if check=True and rc != 0.
	if checkFlag && rc != 0 {
		exc := object.NewException(calledProcErr,
			fmt.Sprintf("Command returned non-zero exit status %d.", rc))
		exc.Dict = object.NewDict()
		exc.Dict.SetStr("returncode", object.NewInt(rc))
		exc.Dict.SetStr("cmd", a[0])
		exc.Dict.SetStr("output", stdoutVal)
		exc.Dict.SetStr("stdout", stdoutVal)
		exc.Dict.SetStr("stderr", stderrVal)
		return nil, exc
	}

	// Build CompletedProcess.
	return i.spCompletedProcess(a[0], rc, stdoutVal, stderrVal, completedProcCls, calledProcErr), nil
}

// ─── CompletedProcess ────────────────────────────────────────────────────────

func (i *Interp) spCompletedProcess(
	argsObj object.Object, rc int64,
	stdout, stderr object.Object,
	completedProcCls, calledProcErr *object.Class,
) *object.Instance {
	cls := completedProcCls
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	inst.Dict.SetStr("args",       argsObj)
	inst.Dict.SetStr("returncode", object.NewInt(rc))
	inst.Dict.SetStr("stdout",     stdout)
	inst.Dict.SetStr("stderr",     stderr)

	// check_returncode on instance so each CP captures its own rc/argsObj.
	inst.Dict.SetStr("check_returncode", &object.BuiltinFunc{Name: "check_returncode",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if rc != 0 {
				exc := object.NewException(calledProcErr,
					fmt.Sprintf("Command returned non-zero exit status %d.", rc))
				exc.Dict = object.NewDict()
				exc.Dict.SetStr("returncode", object.NewInt(rc))
				exc.Dict.SetStr("cmd",        argsObj)
				exc.Dict.SetStr("output",     stdout)
				exc.Dict.SetStr("stdout",     stdout)
				exc.Dict.SetStr("stderr",     stderr)
				return nil, exc
			}
			return object.None, nil
		}})

	return inst
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// spBuildArgv converts a Python args object to a Go string slice.
func spBuildArgv(argsObj object.Object, shell bool) ([]string, error) {
	switch t := argsObj.(type) {
	case *object.Str:
		if shell {
			return []string{t.V}, nil
		}
		// Single string without shell: split by spaces (simple heuristic)
		parts := strings.Fields(t.V)
		if len(parts) == 0 {
			return nil, fmt.Errorf("empty command string")
		}
		return parts, nil
	case *object.List:
		return spObjsToStrings(t.V)
	case *object.Tuple:
		return spObjsToStrings(t.V)
	}
	return nil, fmt.Errorf("unsupported args type %T", argsObj)
}

func spObjsToStrings(objs []object.Object) ([]string, error) {
	out := make([]string, 0, len(objs))
	for _, o := range objs {
		switch t := o.(type) {
		case *object.Str:
			out = append(out, t.V)
		case *object.Bytes:
			out = append(out, string(t.V))
		case *object.Int:
			if t.IsInt64() {
				out = append(out, fmt.Sprintf("%d", t.Int64()))
			}
		default:
			out = append(out, object.Repr(o))
		}
	}
	return out, nil
}

// spDictToEnv converts a Python dict to []string{"KEY=VALUE", ...}.
func spDictToEnv(d *object.Dict) []string {
	ks, vs := d.Items()
	env := make([]string, 0, len(ks))
	for idx, k := range ks {
		var key, val string
		if s, ok := k.(*object.Str); ok {
			key = s.V
		}
		if s, ok := vs[idx].(*object.Str); ok {
			val = s.V
		}
		env = append(env, key+"="+val)
	}
	return env
}

// spMakeWriteStream wraps an io.WriteCloser as a Python-level file-like object.
func spMakeWriteStream(wc io.WriteCloser) *object.Instance {
	cls := &object.Class{Name: "WriteStream", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	cls.Dict.SetStr("write", &object.BuiltinFunc{Name: "write",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return object.NewInt(0), nil
			}
			var data []byte
			switch t := a[0].(type) {
			case *object.Bytes:
				data = t.V
			case *object.Str:
				data = []byte(t.V)
			}
			n, err := wc.Write(data)
			if err != nil {
				return nil, object.Errorf(nil, "write: %v", err) //nolint
			}
			return object.NewInt(int64(n)), nil
		}})
	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			wc.Close() //nolint
			return object.None, nil
		}})
	return inst
}

// spMakeLazyReader returns a stream object whose read() returns buffered data.
func spMakeLazyReader(ps *spPopenState, isStdout bool) *object.Instance {
	cls := &object.Class{Name: "ReadStream", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	cls.Dict.SetStr("read", &object.BuiltinFunc{Name: "read",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ps.mu.Lock()
			defer ps.mu.Unlock()
			var buf *bytes.Buffer
			if isStdout {
				buf = ps.stdoutBuf
			} else {
				buf = ps.stderrBuf
			}
			if buf == nil {
				return &object.Bytes{V: nil}, nil
			}
			data := append([]byte(nil), buf.Bytes()...)
			if ps.textMode {
				return &object.Str{V: string(data)}, nil
			}
			return &object.Bytes{V: data}, nil
		}})
	return inst
}

// spRuntimeErr is a fallback used in spMakeWriteStream — define a sentinel.
var _ = atomic.Int32{} // ensure sync/atomic imported
