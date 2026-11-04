package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func platformInit() {}

func ccName() string { return "claude" }

func ccPlatform() string {
	if runtime.GOARCH == "arm64" {
		return "darwin-arm64"
	}
	if out, _ := exec.Command("sysctl", "-n", "sysctl.proc_translated").Output(); strings.TrimSpace(string(out)) == "1" {
		return "darwin-arm64"
	}
	return "darwin-x64"
}

func afterDownload(bin string) { exec.Command("xattr", "-d", "com.apple.quarantine", bin).Run() }

func platformDeps() (map[string]string, error) { return nil, nil }

func addToPath(dir string) {
	for _, rc := range []string{".zshrc", ".bash_profile"} {
		p := filepath.Join(home(), rc)
		b, _ := os.ReadFile(p)
		if strings.Contains(string(b), ".local/bin") {
			continue
		}
		if f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintf(f, "\n# AI 課程安裝程式加的：Claude Code\nexport PATH=\"$HOME/.local/bin:$PATH\"\n")
			f.Close()
		}
	}
	os.Setenv("PATH", dir+":"+os.Getenv("PATH"))
}

func openURL(u string) { _ = exec.Command("open", u).Start() }

const dialogAS = `set h to system attribute "AIC_HINT"
try
	set r to display dialog h default answer "" with title "AI 課程：填入 Z.AI 金鑰" buttons {"取消", "打開金鑰網頁", "確定"} default button "確定" cancel button "取消"
on error
	return "CANCEL"
end try
if button returned of r is "打開金鑰網頁" then
	open location (system attribute "AIC_URL")
	return "AGAIN"
end if
return "KEY:" & text returned of r`

func keyDialog(hint string) (string, bool) {
	for i := 0; i < 5; i++ {
		cmd := exec.Command("osascript", "-e", dialogAS)
		cmd.Env = append(os.Environ(), "AIC_HINT="+hint, "AIC_URL="+keyPage)
		b, err := cmd.Output()
		if err != nil {
			return "", false
		}
		out := strings.TrimSpace(string(b))
		switch {
		case strings.HasPrefix(out, "CANCEL"):
			fmt.Println("        你按了取消。之後再點一次這個安裝程式就可以填。")
			pause()
			os.Exit(2)
		case strings.HasPrefix(out, "AGAIN"):
			continue
		case strings.HasPrefix(out, "KEY:"):
			return out[4:], true
		}
	}
	return "", false
}

func makeLauncher(cc, work string) string {
	desk := filepath.Join(home(), "Desktop")
	if _, err := os.Stat(desk); err != nil {
		return ""
	}
	p := filepath.Join(desk, "開始上課.command")
	if os.WriteFile(p, []byte("#!/bin/zsh\ncd \""+work+"\" && exec \""+cc+"\"\n"), 0o755) != nil {
		return ""
	}
	return p
}
