//go:build unix

package vm

import (
	"os"
	"syscall"

	"github.com/tamnd/goipy/object"
)

func osApplySysStat(inst *object.Instance, info os.FileInfo) {
	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		inst.Dict.SetStr("st_mode", object.NewInt(int64(sys.Mode)))
		inst.Dict.SetStr("st_ino", object.NewInt(int64(sys.Ino)))
		inst.Dict.SetStr("st_dev", object.NewInt(int64(sys.Dev)))
		inst.Dict.SetStr("st_nlink", object.NewInt(int64(sys.Nlink)))
		inst.Dict.SetStr("st_uid", object.NewInt(int64(sys.Uid)))
		inst.Dict.SetStr("st_gid", object.NewInt(int64(sys.Gid)))
		atimeSec, atimeNsec := statAtime(sys)
		ctimeSec, ctimeNsec := statCtime(sys)
		inst.Dict.SetStr("st_atime", &object.Float{V: float64(atimeSec) + float64(atimeNsec)/1e9})
		inst.Dict.SetStr("st_ctime", &object.Float{V: float64(ctimeSec) + float64(ctimeNsec)/1e9})
	} else {
		inst.Dict.SetStr("st_mode", object.NewInt(goModeToPosix(info.Mode())))
	}
}

func osInodeFromInfo(info os.FileInfo) int64 {
	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		return int64(sys.Ino)
	}
	return 0
}

func osFileCrossDevice(info1, info2 os.FileInfo) bool {
	s1, ok1 := info1.Sys().(*syscall.Stat_t)
	s2, ok2 := info2.Sys().(*syscall.Stat_t)
	if !ok1 || !ok2 {
		return false
	}
	return s1.Dev != s2.Dev
}
