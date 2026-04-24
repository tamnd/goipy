package vm

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// unameCacheOnce guards lazy population of unameData.
var (
	unameCacheOnce sync.Once
	unameData      struct {
		system, node, release, version, machine, processor string
	}
)

// loadUname populates unameData by calling uname sub-commands once.
func loadUname() {
	unameCacheOnce.Do(func() {
		run := func(args ...string) string {
			out, err := exec.Command("uname", args...).Output()
			if err != nil {
				return ""
			}
			return strings.TrimSpace(string(out))
		}

		d := &unameData
		d.system = run("-s")
		d.node = run("-n")
		d.release = run("-r")
		d.version = run("-v")
		d.machine = run("-m")
		// -p is not available on all platforms; fall back to machine.
		d.processor = run("-p")
		if d.processor == "unknown" {
			d.processor = d.machine
		}

		// Fallbacks using runtime when uname is unavailable (Windows, etc.).
		if d.system == "" {
			d.system = goosToSystem(runtime.GOOS)
		}
		if d.machine == "" {
			d.machine = goarchToMachine(runtime.GOARCH)
		}
		if d.processor == "" {
			d.processor = d.machine
		}
		if d.node == "" {
			d.node, _ = os.Hostname()
		}
	})
}

// goosToSystem maps GOOS strings to Python platform.system() equivalents.
func goosToSystem(goos string) string {
	switch goos {
	case "darwin":
		return "Darwin"
	case "linux":
		return "Linux"
	case "windows":
		return "Windows"
	case "freebsd":
		return "FreeBSD"
	case "openbsd":
		return "OpenBSD"
	case "netbsd":
		return "NetBSD"
	default:
		return strings.Title(goos) //nolint:staticcheck
	}
}

// goarchToMachine maps GOARCH strings to Python platform.machine() equivalents.
func goarchToMachine(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	case "386":
		return "i386"
	case "arm64":
		return "arm64"
	case "arm":
		return "armv7l"
	default:
		return arch
	}
}

// newUnamedResult builds an *object.Instance holding the uname fields.
func newUnameResult(cls *object.Class) *object.Instance {
	loadUname()
	d := &unameData
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("system", &object.Str{V: d.system})
	inst.Dict.SetStr("node", &object.Str{V: d.node})
	inst.Dict.SetStr("release", &object.Str{V: d.release})
	inst.Dict.SetStr("version", &object.Str{V: d.version})
	inst.Dict.SetStr("machine", &object.Str{V: d.machine})
	inst.Dict.SetStr("processor", &object.Str{V: d.processor})
	return inst
}

// freedesktopOsRelease reads /etc/os-release and returns a dict.
func freedesktopOsRelease() *object.Dict {
	d := object.NewDict()
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return d
	}
	defer f.Close() //nolint
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := line[:idx]
		val := line[idx+1:]
		// strip surrounding quotes
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		d.SetStr(key, &object.Str{V: val})
	}
	return d
}

func (i *Interp) buildPlatform() *object.Module {
	m := &object.Module{Name: "platform", Dict: object.NewDict()}

	// Build a minimal uname_result class so isinstance checks work.
	unameCls := &object.Class{Name: "uname_result", Dict: object.NewDict()}

	// --- machine() ---
	m.Dict.SetStr("machine", &object.BuiltinFunc{Name: "machine", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		loadUname()
		return &object.Str{V: unameData.machine}, nil
	}})

	// --- processor() ---
	m.Dict.SetStr("processor", &object.BuiltinFunc{Name: "processor", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		loadUname()
		return &object.Str{V: unameData.processor}, nil
	}})

	// --- node() ---
	m.Dict.SetStr("node", &object.BuiltinFunc{Name: "node", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		loadUname()
		return &object.Str{V: unameData.node}, nil
	}})

	// --- system() ---
	m.Dict.SetStr("system", &object.BuiltinFunc{Name: "system", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		loadUname()
		return &object.Str{V: unameData.system}, nil
	}})

	// --- release() ---
	m.Dict.SetStr("release", &object.BuiltinFunc{Name: "release", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		loadUname()
		return &object.Str{V: unameData.release}, nil
	}})

	// --- version() ---
	m.Dict.SetStr("version", &object.BuiltinFunc{Name: "version", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		loadUname()
		return &object.Str{V: unameData.version}, nil
	}})

	// --- platform(aliased=False, terse=False) ---
	m.Dict.SetStr("platform", &object.BuiltinFunc{Name: "platform", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		loadUname()
		d := &unameData
		result := fmt.Sprintf("%s-%s-%s-%s", d.system, d.machine, d.release, d.version)
		// terse: omit the version string
		terse := false
		if len(args) >= 2 {
			terse = object.Truthy(args[1])
		}
		if kw != nil {
			if v, ok := kw.GetStr("terse"); ok {
				terse = object.Truthy(v)
			}
		}
		if terse {
			result = fmt.Sprintf("%s-%s-%s", d.system, d.machine, d.release)
		}
		return &object.Str{V: result}, nil
	}})

	// --- architecture(executable='', bits='', linkage='') ---
	m.Dict.SetStr("architecture", &object.BuiltinFunc{Name: "architecture", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		bits := "64bit"
		if runtime.GOARCH == "386" || runtime.GOARCH == "arm" {
			bits = "32bit"
		}
		linkage := ""
		switch runtime.GOOS {
		case "linux":
			linkage = "ELF"
		case "darwin":
			linkage = "Mach-O"
		case "windows":
			linkage = "WindowsPE"
		}
		return &object.Tuple{V: []object.Object{
			&object.Str{V: bits},
			&object.Str{V: linkage},
		}}, nil
	}})

	// --- Python version info ---
	m.Dict.SetStr("python_version", &object.BuiltinFunc{Name: "python_version", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "3.14.0"}, nil
	}})

	m.Dict.SetStr("python_version_tuple", &object.BuiltinFunc{Name: "python_version_tuple", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "3"},
			&object.Str{V: "14"},
			&object.Str{V: "0"},
		}}, nil
	}})

	m.Dict.SetStr("python_implementation", &object.BuiltinFunc{Name: "python_implementation", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "CPython"}, nil
	}})

	m.Dict.SetStr("python_build", &object.BuiltinFunc{Name: "python_build", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "goipy"},
			&object.Str{V: "Apr 2026"},
		}}, nil
	}})

	m.Dict.SetStr("python_compiler", &object.BuiltinFunc{Name: "python_compiler", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "GCC"}, nil
	}})

	m.Dict.SetStr("python_branch", &object.BuiltinFunc{Name: "python_branch", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "main"}, nil
	}})

	m.Dict.SetStr("python_revision", &object.BuiltinFunc{Name: "python_revision", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: ""}, nil
	}})

	// --- uname() ---
	m.Dict.SetStr("uname", &object.BuiltinFunc{Name: "uname", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return newUnameResult(unameCls), nil
	}})

	// --- Windows stubs ---
	m.Dict.SetStr("win32_ver", &object.BuiltinFunc{Name: "win32_ver", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		release := &object.Str{V: ""}
		version := &object.Str{V: ""}
		csd := &object.Str{V: ""}
		ptype := &object.Str{V: ""}
		if runtime.GOOS == "windows" {
			loadUname()
			release = &object.Str{V: unameData.release}
			version = &object.Str{V: unameData.version}
		}
		return &object.Tuple{V: []object.Object{release, version, csd, ptype}}, nil
	}})

	m.Dict.SetStr("win32_edition", &object.BuiltinFunc{Name: "win32_edition", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("win32_is_iot", &object.BuiltinFunc{Name: "win32_is_iot", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})

	// --- macOS stub ---
	m.Dict.SetStr("mac_ver", &object.BuiltinFunc{Name: "mac_ver", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		rel := ""
		machine := ""
		if runtime.GOOS == "darwin" {
			loadUname()
			rel = unameData.release
			machine = unameData.machine
		}
		versioninfo := &object.Tuple{V: []object.Object{
			&object.Str{V: ""},
			&object.Str{V: ""},
			&object.Str{V: ""},
		}}
		return &object.Tuple{V: []object.Object{
			&object.Str{V: rel},
			versioninfo,
			&object.Str{V: machine},
		}}, nil
	}})

	// --- Linux-specific: freedesktop_os_release() ---
	m.Dict.SetStr("freedesktop_os_release", &object.BuiltinFunc{Name: "freedesktop_os_release", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return freedesktopOsRelease(), nil
	}})

	// Expose the uname_result class so tests can do isinstance checks if needed.
	m.Dict.SetStr("uname_result", unameCls)

	return m
}
