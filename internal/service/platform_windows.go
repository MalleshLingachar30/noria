//go:build windows

package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/amxv/noria/internal/config"
)

func configureDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008 | 0x00000200}
}
func startPlatform(home, exe string, c config.Config) error { return detach(home, exe, c) }
func stopPlatform(home string) error                        { return nil }
func loginPlatform(home, exe string, enabled bool) error {
	label := Label(home)
	if !enabled {
		_ = command("schtasks.exe", "/Delete", "/TN", label, "/F")
		return nil
	}
	// InteractiveToken runs only as this signed-in user, with no stored password.
	xml := `<?xml version="1.0" encoding="UTF-16"?><Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task"><Triggers><LogonTrigger><Enabled>true</Enabled></LogonTrigger></Triggers><Principals><Principal id="Author"><LogonType>InteractiveToken</LogonType><RunLevel>LeastPrivilege</RunLevel></Principal></Principals><Settings><MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy><DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries><StopIfGoingOnBatteries>false</StopIfGoingOnBatteries><ExecutionTimeLimit>PT0S</ExecutionTimeLimit></Settings><Actions Context="Author"><Exec><Command>` + quoteXML(exe) + `</Command><Arguments>start --home ` + quoteXML(`"`+home+`"`) + `</Arguments><WorkingDirectory>` + quoteXML(home) + `</WorkingDirectory></Exec></Actions></Task>`
	// schtasks accepts UTF-8 when declared as UTF-8.
	xml = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>" + xml[len(`<?xml version="1.0" encoding="UTF-16"?>`):]
	path := filepath.Join(home, "login-task.xml")
	if e := os.WriteFile(path, []byte(xml), 0600); e != nil {
		return e
	}
	return command("schtasks.exe", "/Create", "/TN", label, "/XML", path, "/F")
}
