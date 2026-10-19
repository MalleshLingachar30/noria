// Package host contains OS glue only. Policy and preferences live in the Go CLI.
package host

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"

	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func Run(exe string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, exe, args...)
	if b, e := c.CombinedOutput(); e != nil {
		return fmt.Errorf("%s failed: %s", filepath.Base(exe), strings.TrimSpace(string(b)))
	}
	return nil
}
func X(s string) string { var b bytes.Buffer; _ = xml.EscapeText(&b, []byte(s)); return b.String() }
func OpenURL(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return Run("/usr/bin/open", url)
	case "windows":
		return Run("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return Run("xdg-open", url)
	}
}
