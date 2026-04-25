//go:build windows

package vm

import (
	"os"

	"github.com/tamnd/goipy/object"
)

func osApplySysStat(inst *object.Instance, info os.FileInfo) {
	inst.Dict.SetStr("st_mode", object.NewInt(goModeToPosix(info.Mode())))
}

func osInodeFromInfo(_ os.FileInfo) int64 { return 0 }

func osFileCrossDevice(_, _ os.FileInfo) bool { return false }

func osAtime(info os.FileInfo) float64 {
	return float64(info.ModTime().UnixNano()) / 1e9
}

func osCtime(info os.FileInfo) float64 {
	return float64(info.ModTime().UnixNano()) / 1e9
}
