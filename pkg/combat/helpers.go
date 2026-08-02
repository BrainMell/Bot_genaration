package combat

import (
        "image"
        "image/png"
        "io"
        "os/exec"
        "time"
)

// newExecCommand wraps exec.Command so it can be stubbed in tests.
func newExecCommand(name string, args ...string) *exec.Cmd {
        return exec.Command(name, args...)
}

// newPngEncode wraps png.Encode so it can be stubbed in tests.
func newPngEncode(w io.Writer, img image.Image) error {
        return png.Encode(w, img)
}

// timeNow returns the current time — wrapped so it can be stubbed in tests.
func timeNow() time.Time {
        return time.Now()
}
