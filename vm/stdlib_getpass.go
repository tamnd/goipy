package vm

import (
	"os"
	"os/user"

	"github.com/tamnd/goipy/object"
)

// getpassCurrentUser returns the login name via env vars first, then os/user.
func getpassCurrentUser() string {
	for _, env := range []string{"LOGNAME", "USER", "LNAME", "USERNAME"} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return ""
}

// buildGetpass constructs the getpass module:
//   - getpass(prompt='Password: ', stream=None) -> str  (returns "" in non-interactive mode)
//   - getuser() -> str
//   - GetPassWarning  (UserWarning subclass)
func (i *Interp) buildGetpass() *object.Module {
	m := &object.Module{Name: "getpass", Dict: object.NewDict()}

	// GetPassWarning — subclass of UserWarning.
	// UserWarning is registered in builtins; look it up so we can set the base.
	var userWarningCls *object.Class
	if v, ok := i.Builtins.GetStr("UserWarning"); ok {
		if cls, ok2 := v.(*object.Class); ok2 {
			userWarningCls = cls
		}
	}
	var warnBases []*object.Class
	if userWarningCls != nil {
		warnBases = []*object.Class{userWarningCls}
	}
	warnCls := &object.Class{Name: "GetPassWarning", Bases: warnBases, Dict: object.NewDict()}
	m.Dict.SetStr("GetPassWarning", warnCls)

	// getpass(prompt='Password: ', stream=None, *, echo_char=None) -> str
	getpassFn := &object.BuiltinFunc{
		Name: "getpass",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// keyword arg takes precedence over positional
			if kw != nil {
				if v, ok := kw.GetStr("prompt"); ok {
					_ = v // accepted but ignored in non-interactive mode
				}
			}
			// positional arg[0] is prompt — accepted but ignored
			return &object.Str{V: ""}, nil
		},
	}
	m.Dict.SetStr("getpass", getpassFn)

	// getuser() -> str
	getUserFn := &object.BuiltinFunc{
		Name: "getuser",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			name := getpassCurrentUser()
			return &object.Str{V: name}, nil
		},
	}
	m.Dict.SetStr("getuser", getUserFn)

	return m
}
