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
	shown := dir
	if rel, err := filepath.Rel(home(), dir); err == nil && !strings.HasPrefix(rel, "..") {
		shown = "$HOME/" + rel
	}
	for _, rc := range []string{".zshrc", ".bash_profile"} {
		p := filepath.Join(home(), rc)
		b, _ := os.ReadFile(p)
		if strings.Contains(string(b), shown) {
			continue
		}
		if f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintf(f, "\n# AI 課程安裝程式加的\nexport PATH=\"%s:$PATH\"\n", shown)
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

func ocBinDir() string { return filepath.Join(home(), ".opencode", "bin") }
func ocName() string   { return "opencode" }
func ocPlatform() string {
	if strings.HasSuffix(ccPlatform(), "arm64") {
		return "darwin-arm64"
	}
	return "darwin-x64"
}

const toolAS = `try
	set r to choose from list {(system attribute "AIC_A"), (system attribute "AIC_B")} with title "AI 課程：選要用的工具" with prompt "老師沒特別說，就直接按 Enter（用 OpenCode）。" default items {(system attribute "AIC_A")} OK button name "下一步" cancel button name "取消"
on error
	return "CANCEL"
end try
if r is false then return "CANCEL"
if (item 1 of r) starts with "②" then return "TOOL:claude"
return "TOOL:opencode"`

func toolDialog(a, b string) (string, bool) {
	cmd := exec.Command("osascript", "-e", toolAS)
	cmd.Env = append(os.Environ(), "AIC_A="+a, "AIC_B="+b)
	o, err := cmd.Output()
	if err != nil {
		return "", false
	}
	out := string(o)
	switch {
	case strings.Contains(out, "CANCEL"):
		fmt.Println("        你按了取消。之後再點一次這個安裝程式就可以裝。")
		pause()
		os.Exit(2)
	case strings.Contains(out, "TOOL:opencode"):
		return toolOC, true
	case strings.Contains(out, "TOOL:claude"):
		return toolCC, true
	}
	return "", false
}

func desktopDirs() []string { return []string{filepath.Join(home(), "Desktop")} }

func makeLauncher(exe, work, name string) string {
	desk := desktopDirs()[0]
	if _, err := os.Stat(desk); err != nil {
		return ""
	}
	p := filepath.Join(desk, name+".command")
	if os.WriteFile(p, []byte("#!/bin/zsh\ncd \""+work+"\" && exec \""+exe+"\"\n"), 0o755) != nil {
		return ""
	}
	return p
}
