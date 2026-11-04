package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	gitVer   = "2.56.0.windows.4"
	gitX64   = "https://github.com/git-for-windows/git/releases/download/v2.56.0.windows.4/PortableGit-2.56.0.4-64-bit.7z.exe"
	gitArm64 = "https://github.com/git-for-windows/git/releases/download/v2.56.0.windows.4/PortableGit-2.56.0.4-arm64.7z.exe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

func platformInit() {
	kernel32.NewProc("SetConsoleOutputCP").Call(65001)
	kernel32.NewProc("SetConsoleCP").Call(65001)
	kernel32.NewProc("SetConsoleTitleW").Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("AI 課程一鍵安裝"))))
}

func ccName() string { return "claude.exe" }

func nativeARM() bool {
	if runtime.GOARCH == "arm64" {
		return true
	}
	p := kernel32.NewProc("IsWow64Process2")
	if p.Find() == nil {
		var proc, native uint16
		h, _ := syscall.GetCurrentProcess()
		r, _, _ := p.Call(uintptr(h), uintptr(unsafe.Pointer(&proc)), uintptr(unsafe.Pointer(&native)))
		return r != 0 && native == 0xAA64
	}
	return false
}

func ccPlatform() string {
	if nativeARM() {
		return "win32-arm64"
	}
	return "win32-x64"
}

func afterDownload(bin string) {}

func localAppData() string {
	if *flagHome != "" {
		return filepath.Join(*flagHome, "AppData", "Local")
	}
	if v := os.Getenv("LOCALAPPDATA"); v != "" {
		return v
	}
	return filepath.Join(home(), "AppData", "Local")
}

// Claude Code 在 Windows 要 Git Bash：已經有就用，沒有就免管理員裝 PortableGit
func platformDeps() (map[string]string, error) {
	cands := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "bin", "bash.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Git", "bin", "bash.exe"),
		filepath.Join(localAppData(), "Programs", "Git", "bin", "bash.exe"),
	}
	if *flagHome != "" {
		cands = nil // 測試：當成什麼都沒裝
	}
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			fmt.Printf("        ✓ 找到 Git Bash：%s\n", c)
			return map[string]string{"CLAUDE_CODE_GIT_BASH_PATH": c}, nil
		}
	}
	dir := filepath.Join(localAppData(), "Programs", "ai-class", "PortableGit")
	bash := filepath.Join(dir, "bin", "bash.exe")
	if _, err := os.Stat(bash); err != nil {
		fmt.Println("        安裝 Git Bash（Claude Code 在 Windows 需要，約 60 MB）")
		url := gitX64
		if nativeARM() {
			url = gitArm64
		}
		tmp := filepath.Join(os.TempDir(), "aic-portablegit.7z.exe")
		if err := download(url, tmp); err != nil {
			return nil, fmt.Errorf("Git 下載失敗（%v）：檢查網路後再點一次", err)
		}
		defer os.Remove(tmp)
		os.MkdirAll(dir, 0o755)
		cmd := exec.Command(tmp, "-o"+dir, "-y")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("Git 解壓失敗：%v %s", err, lastLine(string(out)))
		}
		if _, err := os.Stat(bash); err != nil {
			return nil, fmt.Errorf("Git 解壓後找不到 bash.exe")
		}
	}
	fmt.Printf("        ✓ Git Bash：%s\n", bash)
	return map[string]string{"CLAUDE_CODE_GIT_BASH_PATH": bash}, nil
}

func ps(script string, env ...string) (string, error) {
	u := utf16.Encode([]rune("[Console]::OutputEncoding=[Text.Encoding]::UTF8\n" + script))
	b := make([]byte, len(u)*2)
	for i, c := range u {
		b[2*i], b[2*i+1] = byte(c), byte(c>>8)
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-STA", "-EncodedCommand", base64.StdEncoding.EncodeToString(b))
	cmd.Env = append(os.Environ(), env...)
	// 不設 HideWindow：會把對話框一起藏起來（子程序共用這個主控台，本來就不會多開視窗）
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func addToPath(dir string) {
	if *flagHome == "" {
		_, _ = ps(`$d=$env:AIC_DIR; $p=[Environment]::GetEnvironmentVariable('Path','User'); if(-not $p){$p=''}
if(($p -split ';') -notcontains $d){ [Environment]::SetEnvironmentVariable('Path', ($d + ';' + $p).TrimEnd(';'), 'User') }`, "AIC_DIR="+dir)
	}
	os.Setenv("PATH", dir+";"+os.Getenv("PATH"))
}

func openURL(u string) { _ = exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start() }

const dialogPS = `Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()
$f=New-Object Windows.Forms.Form; $f.Text='AI 課程：填入 Z.AI 金鑰'; $f.Size=New-Object Drawing.Size(640,300)
$f.StartPosition='CenterScreen'; $f.TopMost=$true; $f.FormBorderStyle='FixedDialog'; $f.MaximizeBox=$false; $f.MinimizeBox=$false
$font=New-Object Drawing.Font('Microsoft JhengHei UI',12)
$l=New-Object Windows.Forms.Label; $l.Text=$env:AIC_HINT; $l.Location='20,18'; $l.Size='590,60'; $l.Font=$font; $f.Controls.Add($l)
$t=New-Object Windows.Forms.TextBox; $t.Location='20,90'; $t.Size='585,30'; $t.Font=New-Object Drawing.Font('Consolas',13); $f.Controls.Add($t)
$k=New-Object Windows.Forms.LinkLabel; $k.Text='還沒有金鑰？點這裡打開 Z.AI 金鑰網頁'; $k.Location='20,140'; $k.Size='450,28'; $k.Font=$font
$k.Add_LinkClicked({ Start-Process $env:AIC_URL }); $f.Controls.Add($k)
$ok=New-Object Windows.Forms.Button; $ok.Text='確定（Enter）'; $ok.Location='440,190'; $ok.Size='165,44'; $ok.Font=$font; $ok.DialogResult='OK'; $f.Controls.Add($ok); $f.AcceptButton=$ok
$no=New-Object Windows.Forms.Button; $no.Text='取消'; $no.Location='320,190'; $no.Size='110,44'; $no.Font=$font; $no.DialogResult='Cancel'; $f.Controls.Add($no); $f.CancelButton=$no
$f.Add_Shown({ $f.Activate(); $t.Focus() })
if($f.ShowDialog() -eq 'OK'){ 'KEY:' + $t.Text } else { 'CANCEL' }`

func keyDialog(hint string) (string, bool) {
	out, err := ps(dialogPS, "AIC_HINT="+hint, "AIC_URL="+keyPage)
	if err != nil {
		return "", false
	}
	if strings.HasPrefix(out, "CANCEL") {
		fmt.Println("        你按了取消。之後再點一次這個安裝程式就可以填。")
		pause()
		os.Exit(2)
	}
	if i := strings.Index(out, "KEY:"); i >= 0 {
		return out[i+4:], true
	}
	return "", false
}

func makeLauncher(cc, work string) string {
	desk := ""
	if *flagHome != "" {
		desk = filepath.Join(*flagHome, "Desktop")
	} else if d, err := ps(`[Environment]::GetFolderPath('Desktop')`); err == nil {
		desk = d
	}
	if desk == "" {
		return ""
	}
	os.MkdirAll(desk, 0o755)
	p := filepath.Join(desk, "開始上課.cmd")
	body := "@echo off\r\nchcp 65001 >nul\r\ntitle 開始上課\r\ncd /d \"" + work + "\"\r\n\"" + cc + "\"\r\n"
	if os.WriteFile(p, []byte(body), 0o644) != nil {
		return ""
	}
	return p
}
