package cdn

import (
	"bytes"
	"os/exec"
)

func gomodcache() string {
	buf := &bytes.Buffer{}
	cmd := exec.Command("go", "env", "GOMODCACHE")
	cmd.Stdout = buf
	_ = cmd.Run()

	out := buf.String()
	return out[:len(out)-1] // cut trailing \n
}
