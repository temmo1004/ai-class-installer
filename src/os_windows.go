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

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

func platformInit() {
	kernel32.NewProc("SetConsoleOutputCP").Call(65001)
	kernel32.NewProc("SetConsoleCP").Call(65001)
	kernel32.NewProc("SetConsoleTitleW").Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("AI 課程一鍵安裝"))))
}

func localAppData() string {
	if *flagHome != "" {
		return filepath.Join(*flagHome, "AppData", "Local")
	}
	if v := os.Getenv("LOCALAPPDATA"); v != "" {
		return v
	}
	return filepath.Join(home(), "AppData", "Local")
}

func binDir() string { return filepath.Join(localAppData(), "Programs", "ai-class") }
func ocName() string { return "opencode.exe" }

// 真正的 CPU：x64 版的安裝程式在 ARM 筆電上是模擬執行，要問系統本機是不是 ARM64
func platformName() string {
	if runtime.GOARCH == "arm64" {
		return "windows-arm64"
	}
	p := kernel32.NewProc("IsWow64Process2")
	if p.Find() == nil {
		var proc, native uint16
		h, _ := syscall.GetCurrentProcess()
		r, _, _ := p.Call(uintptr(h), uintptr(unsafe.Pointer(&proc)), uintptr(unsafe.Pointer(&native)))
		if r != 0 && native == 0xAA64 {
			return "windows-arm64"
		}
	}
	return "windows-x64"
}

func afterInstall(exe string) {}

func ps(script string, env ...string) (string, error) {
	u := utf16.Encode([]rune("[Console]::OutputEncoding=[Text.Encoding]::UTF8\n" + script))
	b := make([]byte, len(u)*2)
	for i, c := range u {
		b[2*i], b[2*i+1] = byte(c), byte(c>>8)
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-STA", "-EncodedCommand", base64.StdEncoding.EncodeToString(b))
	cmd.Env = append(os.Environ(), env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func addToPath(dir string) {
	if *flagHome != "" {
		return // 測試用的假使用者，不動真的 PATH
	}
	_, _ = ps(`$d=$env:AIC_DIR; $p=[Environment]::GetEnvironmentVariable('Path','User'); if(-not $p){$p=''}
if(($p -split ';') -notcontains $d){ [Environment]::SetEnvironmentVariable('Path', ($d + ';' + $p).TrimEnd(';'), 'User') }`, "AIC_DIR="+dir)
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

// 跳出 Windows 視窗問金鑰。ok=false 表示視窗開不了，改在黑視窗問。
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

func makeLauncher(oc, work string) string {
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
	p := filepath.Join(desk, launcherW)
	body := "@echo off\r\nchcp 65001 >nul\r\ntitle OpenCode\r\ncd /d \"" + work + "\"\r\n\"" + oc + "\"\r\n"
	if os.WriteFile(p, []byte(body), 0o644) != nil {
		return ""
	}
	return p
}
