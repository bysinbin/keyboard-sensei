package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	Label         = "com.keyboard.sensei"
	PlistFilename = "com.keyboard.sensei.plist"
)

func GetPlistPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "LaunchAgents", PlistFilename)
}

func GetLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/keyboard-sensei.log"
	}
	logDir := filepath.Join(home, "Library", "Logs")
	_ = os.MkdirAll(logDir, 0755)
	return filepath.Join(logDir, "KeyboardSensei.log")
}

func IsInstalled() bool {
	plist := GetPlistPath()
	if plist == "" {
		return false
	}
	_, err := os.Stat(plist)
	return err == nil
}

func IsRunning() bool {
	cmd := exec.Command("launchctl", "list", Label)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), Label)
}

func Install(execPath string) error {
	if execPath == "" {
		currentExec, err := os.Executable()
		if err != nil {
			return err
		}
		execPath = currentExec
	}

	execPath, err := filepath.Abs(execPath)
	if err != nil {
		return err
	}

	plistPath := GetPlistPath()
	if plistPath == "" {
		return fmt.Errorf("kullanıcı ana dizini bulunamadı")
	}

	_ = os.MkdirAll(filepath.Dir(plistPath), 0755)
	logPath := GetLogPath()

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>-open=false</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>ProcessType</key>
    <string>Interactive</string>
</dict>
</plist>
`, Label, execPath, logPath, logPath)

	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return err
	}

	// Unload first if previously active
	_ = exec.Command("launchctl", "unload", "-w", plistPath).Run()

	// Load service
	cmd := exec.Command("launchctl", "load", "-w", plistPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load başarısız: %v (%s)", err, string(out))
	}

	return nil
}

func Uninstall() error {
	plistPath := GetPlistPath()
	if plistPath == "" {
		return nil
	}

	_ = exec.Command("launchctl", "unload", "-w", plistPath).Run()
	_ = os.Remove(plistPath)
	return nil
}
