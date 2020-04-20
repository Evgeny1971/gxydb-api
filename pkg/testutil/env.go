package testutil

import (
	"path/filepath"
	"runtime"

	"github.com/subosito/gotenv"
)

func init() {
	_, filename, _, _ := runtime.Caller(0)
	rel := filepath.Join(filepath.Dir(filename), "..", "..", ".env")
	gotenv.Load(rel)
}
