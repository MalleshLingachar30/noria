//go:build windows

package tray

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var winMessageBox = windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW")

func showBox(title, text string, flags uintptr) (uintptr, error) {
	t, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return 0, err
	}
	body, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return 0, err
	}
	ret, _, err := winMessageBox.Call(0, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(t)), flags)
	if ret == 0 {
		return 0, err
	}
	return ret, nil
}
func confirm(title, text string) (bool, error) {
	ret, err := showBox(title, text, 0x24) // MB_YESNO | MB_ICONQUESTION
	return ret == 6, err                   // IDYES
}
func message(title, text string) { _, _ = showBox(title, text, 0x40) }

func powershellString(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func powershell(script string) (string, bool, error) {
	// -EncodedCommand prevents shell interpolation of account names and paths.
	chars := utf16.Encode([]rune("[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false);\n" + script))
	raw := make([]byte, len(chars)*2)
	for i, c := range chars {
		raw[2*i] = byte(c)
		raw[2*i+1] = byte(c >> 8)
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-EncodedCommand", base64.StdEncoding.EncodeToString(raw))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	b, err := cmd.Output()
	if err != nil {
		return "", false, fmt.Errorf("Windows input dialog failed: %w", err)
	}
	value := strings.TrimRight(string(b), "\r\n")
	if value == "__NORIA_CANCEL__" {
		return "", false, nil
	}
	return value, true, nil
}
func prompt(title, initial string) (string, bool, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms;
$f = New-Object System.Windows.Forms.Form; $f.Text = ` + powershellString(title) + `; $f.Width=420; $f.Height=160; $f.StartPosition='CenterScreen'; $f.TopMost=$true;
$input = New-Object System.Windows.Forms.TextBox; $input.SetBounds(15,15,375,25); $input.Text = ` + powershellString(initial) + `; $f.Controls.Add($input);
$ok = New-Object System.Windows.Forms.Button; $ok.Text='Save'; $ok.SetBounds(220,60,80,30); $ok.DialogResult='OK'; $f.Controls.Add($ok); $f.AcceptButton=$ok;
$cancel = New-Object System.Windows.Forms.Button; $cancel.Text='Cancel'; $cancel.SetBounds(310,60,80,30); $cancel.DialogResult='Cancel'; $f.Controls.Add($cancel); $f.CancelButton=$cancel;
if ($f.ShowDialog() -eq 'OK') { [Console]::WriteLine($input.Text) } else { [Console]::WriteLine('__NORIA_CANCEL__') }; $f.Dispose()`
	return powershell(script)
}
func chooseFolder() (string, bool, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms;
$f = New-Object System.Windows.Forms.FolderBrowserDialog; $f.Description='Select Codex or Claude Code configuration folder';
if ($f.ShowDialog() -eq 'OK') { [Console]::WriteLine($f.SelectedPath) } else { [Console]::WriteLine('__NORIA_CANCEL__') }; $f.Dispose()`
	return powershell(script)
}
func chooseProvider() (string, bool, error) {
	ret, err := showBox("Choose client", "Configure this folder for Codex (Yes) or Claude Code (No)?", 0x23) // MB_YESNOCANCEL | ICONQUESTION
	if err != nil || ret == 2 {
		return "", false, err
	}
	if ret == 6 {
		return "codex", true, nil
	}
	if ret == 7 {
		return "claude", true, nil
	}
	return "", false, fmt.Errorf("unrecognized provider selection")
}
func openURL(address string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", address)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
