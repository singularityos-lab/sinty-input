package dicttest

import (
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	_, file, _, _ := runtime.Caller(0)
	os.Setenv("SINTY_INPUT_DICT_PATH", filepath.Join(filepath.Dir(file), "..", "..", "..", "dict"))
	os.Setenv("SINTY_INPUT_USER_DIR", filepath.Join(os.TempDir(), "sinty-input-test-none"))
}
