//go:build linux

package tray

import (
	"errors"
	"os/exec"
	"strings"
)

func linuxDialog() string {
	if _, err := exec.LookPath("zenity"); err == nil {
		return "zenity"
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		return "kdialog"
	}
	return ""
}
func dialog(argsZenity, argsKdialog []string) (string, bool, error) {
	tool := linuxDialog()
	if tool == "" {
		return "", false, errors.New("install zenity (GNOME/Xfce/Cinnamon) or kdialog (KDE) for this action")
	}
	args := argsZenity
	if tool == "kdialog" {
		args = argsKdialog
	}
	b, err := exec.Command(tool, args...).Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(string(b)), true, nil
}
func confirm(title, text string) (bool, error) {
	_, ok, err := dialog([]string{"--question", "--title", title, "--text", text, "--no-wrap"}, []string{"--yesno", text, "--title", title})
	return ok, err
}
func prompt(title, initial string) (string, bool, error) {
	return dialog([]string{"--entry", "--title", title, "--text", title, "--entry-text", initial}, []string{"--inputbox", title, initial, "--title", title})
}
func chooseFolder() (string, bool, error) {
	return dialog([]string{"--file-selection", "--directory", "--title", "Select client configuration folder"}, []string{"--getexistingdirectory", ".", "--title", "Select client configuration folder"})
}
func chooseProvider() (string, bool, error) {
	selected, ok, err := dialog([]string{"--list", "--radiolist", "--title", "Choose client", "--column", "Select", "--column", "Client", "TRUE", "codex", "FALSE", "claude"},
		[]string{"--radiolist", "Choose client", "codex", "codex", "on", "claude", "claude", "off"})
	if selected != "codex" && selected != "claude" {
		return "", false, err
	}
	return selected, ok, err
}
func message(title, text string) {
	_, _, _ = dialog([]string{"--info", "--title", title, "--text", text, "--no-wrap"}, []string{"--msgbox", text, "--title", title})
}
func openURL(address string) error {
	cmd := exec.Command("xdg-open", address)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
