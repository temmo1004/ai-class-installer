package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	toolCC   = "claude"
	toolOC   = "opencode"
	ocModel  = "zai/glm-4.5-flash"
	ocCredID = "cred_aiclassinstaller0000000001"
	ocDL     = "https://opencode.ai/update/api/latest/cli/opencode/opencode-"
)

func toolLabel(t string) string {
	if t == toolOC {
		return "OpenCode"
	}
	return "Claude Code"
}

func launcherName(t string) string { return "開始上課（" + toolLabel(t) + "）" }

func ccExe() string { return filepath.Join(home(), ".local", "bin", ccName()) }
func ocExe() string { return filepath.Join(ocBinDir(), ocName()) }

func works(exe string) bool {
	fi, err := os.Stat(exe)
	return err == nil && fi.Size() > 1<<20 && strings.TrimSpace(output(exe, "", "--version")) != ""
}

// 預設工具：--tool 參數 ＞ 安裝程式檔名含 opencode／claude ＞ 跳選擇視窗（預設 Claude Code）
func presetTool() string {
	pick := func(s string) string {
		s = strings.ToLower(s)
		switch {
		case strings.Contains(s, "opencode"):
			return toolOC
		case strings.Contains(s, "claude"):
			return toolCC
		}
		return ""
	}
	if t := pick(*flagTool); t != "" {
		return t
	}
	exe, _ := os.Executable()
	return pick(filepath.Base(exe))
}

func chooseTool(ccOK, ocOK bool) string {
	if t := presetTool(); t != "" {
		fmt.Printf("        已指定：%s\n", toolLabel(t))
		return t
	}
	mark := func(ok bool) string {
		if ok {
			return "　✓ 已安裝"
		}
		return ""
	}
	a := "① Claude Code（推薦）" + mark(ccOK)
	b := "② OpenCode（比較快）" + mark(ocOK)
	if r, ok := toolDialog(a, b); ok {
		return r
	}
	fmt.Printf("        1. %s\n        2. %s\n        選 1 或 2（直接按 Enter = 1）：", a, b)
	s, _ := stdin.ReadString('\n')
	if strings.TrimSpace(s) == "2" {
		return toolOC
	}
	return toolCC
}

func run() error {
	ccOK, ocOK := works(ccExe()), works(ocExe())
	fmt.Println("  【1/4】選工具")
	tool := chooseTool(ccOK, ocOK)
	fmt.Printf("        ✓ %s\n\n", toolLabel(tool))

	fmt.Printf("  【2/4】安裝 %s\n", toolLabel(tool))
	var extra map[string]string
	var err error
	if tool == toolCC {
		fmt.Println("        第一次要下載約 120 MB，請不要關掉這個視窗")
		if _, err = installClaude(); err != nil {
			return err
		}
		ccOK = true
		fmt.Printf("        ✓ 裝好了（%s）\n", strings.TrimSpace(output(ccExe(), "", "--version")))
	} else {
		if _, err = installOpenCode(); err != nil {
			return err
		}
		ocOK = true
		fmt.Printf("        ✓ 裝好了（OpenCode %s）\n", strings.TrimSpace(output(ocExe(), "", "--version")))
	}
	if ccOK { // Windows：Claude Code 要 Git Bash
		if extra, err = platformDeps(); err != nil {
			return err
		}
	}
	work := filepath.Join(home(), workName)
	if err := os.MkdirAll(work, 0o755); err != nil {
		return fmt.Errorf("建不了資料夾 %s：%v", work, err)
	}
	writeAgentNotes(work)
	fmt.Printf("        ✓ 工作資料夾：%s\n\n", work)

	fmt.Println("  【3/4】填 Z.AI 金鑰")
	settings := filepath.Join(home(), ".claude", "settings.json")
	old := currentKey(settings) // Claude Code 的金鑰讀得回來；OpenCode 的存在它自己的資料庫裡讀不回來
	ocKeyed := ocMarkerOK()
	// 可以按 Enter 沿用：這次選的工具已經有金鑰（或 Claude Code 的金鑰可以拿來給 OpenCode 用）
	canKeep := old != "" || (tool == toolOC && ocKeyed)
	key, err := askKey(canKeep)
	if err != nil {
		return err
	}
	if key == "" {
		fmt.Println("        ✓ 沿用原本的金鑰")
	}
	// 兩邊同步：有新金鑰就兩個都換；沒新金鑰就把讀得到的那把補給還沒設好的
	if ccOK {
		k := key
		if k == "" {
			k = old
		}
		if k != "" {
			if err := writeSettings(settings, k, extra); err != nil {
				return fmt.Errorf("寫不進 Claude Code 設定：%v", err)
			}
			if err := skipOnboarding(work); err != nil {
				return fmt.Errorf("寫不進 Claude Code 設定：%v", err)
			}
		}
	}
	if ocOK {
		k := key
		if k == "" && !ocKeyed {
			k = old
		}
		if err := setupOpenCode(work, k); err != nil {
			return err
		}
	}
	fmt.Printf("        ✓ 接好 Z.AI，模型：%s\n\n", glmModel)

	fmt.Println("  【4/4】測試：請 AI 回一句話（約 10–60 秒）")
	if !*flagNoTest {
		t0 := time.Now()
		var out string
		if tool == toolCC {
			out, err = runTestCC(ccExe(), work)
		} else {
			out, err = runTestOC(ocExe(), work)
		}
		if err != nil {
			fmt.Printf("        ! 測試沒過：%v\n        先繼續，打開後問它一句話試試。\n", err)
		} else {
			fmt.Printf("        ✓ AI 回：%s（%.0f 秒）\n", out, time.Since(t0).Seconds())
		}
	}
	removeOldLaunchers()
	if ccOK {
		makeLauncher(ccExe(), work, launcherName(toolCC))
	}
	if ocOK {
		makeLauncher(ocExe(), work, launcherName(toolOC))
	}
	fmt.Printf("\n  全部好了！之後要再開：點桌面的「%s」\n", launcherName(tool))
	fmt.Println("  要換金鑰、或改裝另一個工具：再點一次這個安裝程式")
	if *flagNoStart {
		return nil
	}
	fmt.Printf("\n  正在打開 %s …\n", toolLabel(tool))
	time.Sleep(1500 * time.Millisecond)
	exe := ccExe()
	if tool == toolOC {
		exe = ocExe()
	}
	cmd := exec.Command(exe)
	cmd.Dir = work
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = cleanEnv()
	return cmd.Run()
}

func writeAgentNotes(work string) {
	note := "# AI 課程\n\n- 一律用繁體中文（台灣用語）回答。\n- 使用者是第一次接觸程式的學生：步驟講清楚、一次一步、少用術語。\n- 打開網頁或檔案：Mac 用 `open 檔名`，Windows 用 `start 檔名`。\n- 指令一次只下一個，不要用 && 或 ; 串在一起。\n"
	for _, n := range []string{"CLAUDE.md", "AGENTS.md"} { // Claude Code 讀 CLAUDE.md、OpenCode 讀 AGENTS.md
		p := filepath.Join(work, n)
		if _, err := os.Stat(p); err != nil {
			os.WriteFile(p, []byte(note), 0o644)
		}
	}
}

// 舊版（v1／v2.0–2.1）留在桌面的捷徑換成新名字
func removeOldLaunchers() {
	for _, d := range desktopDirs() {
		for _, n := range []string{"開始上課", "開始上課 OpenCode"} {
			for _, ext := range []string{".cmd", ".command"} {
				p := filepath.Join(d, n+ext)
				if b, err := os.ReadFile(p); err == nil && bytes.Contains(b, []byte(workName)) {
					os.Remove(p)
				}
			}
		}
	}
}

// ---------- OpenCode（v1 的流程） ----------

func installOpenCode() (string, error) {
	exe := ocExe()
	if works(exe) {
		fmt.Println("        已經裝過了，略過下載")
		addToPath(ocBinDir())
		return exe, nil
	}
	if err := os.MkdirAll(ocBinDir(), 0o755); err != nil {
		return "", fmt.Errorf("建不了 %s：%v", ocBinDir(), err)
	}
	fmt.Println("        下載中（約 100–130 MB，網路慢要幾分鐘，請不要關掉這個視窗）")
	tmp := filepath.Join(os.TempDir(), "ai-class-opencode.zip")
	if err := download(ocDL+ocPlatform()+".zip", tmp); err != nil {
		return "", fmt.Errorf("下載失敗（%v）：檢查網路後再點一次", err)
	}
	defer os.Remove(tmp)
	if err := unzipOne(tmp, ocName(), exe); err != nil {
		return "", fmt.Errorf("解壓縮失敗：%v", err)
	}
	afterDownload(exe)
	if !works(exe) {
		return "", errors.New("裝好了但打不開 opencode（防毒軟體擋住？）")
	}
	addToPath(ocBinDir())
	return exe, nil
}

func unzipOne(zipPath, name, dst string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if strings.EqualFold(filepath.Base(f.Name), name) && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			tmp := dst + ".new"
			out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, rc); err != nil {
				out.Close()
				return err
			}
			out.Close()
			os.Remove(dst)
			return os.Rename(tmp, dst)
		}
	}
	return fmt.Errorf("壓縮檔裡沒有 %s", name)
}

func ocConfigDir() string {
	return strings.TrimSpace(output(ocExe(), filepath.Join(home(), workName), "debug", "paths", "config"))
}

func ocMarkerOK() bool {
	if !works(ocExe()) {
		return false
	}
	d := ocConfigDir()
	if d == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(d, "ai-class-installer.json"))
	return err == nil
}

// key 空字串＝不換金鑰，只確保模型設定
func setupOpenCode(work, key string) error {
	cfgDir := ocConfigDir()
	if cfgDir == "" {
		return errors.New("找不到 OpenCode 的設定資料夾")
	}
	if key != "" {
		if err := ocSaveKey(work, key); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return err
	}
	p := filepath.Join(cfgDir, "opencode.json")
	cfg := readJSON(p)
	if _, ok := cfg["$schema"]; !ok {
		cfg["$schema"] = "https://opencode.ai/config.json"
	}
	cfg["model"] = ocModel
	if err := writeJSON(p, cfg, 0o644); err != nil {
		return err
	}
	if key != "" {
		_ = os.WriteFile(filepath.Join(cfgDir, "ai-class-installer.json"), []byte(`{"installer":"`+version+`","model":"`+ocModel+`","at":"`+time.Now().Format(time.RFC3339)+`"}`+"\n"), 0o644)
	}
	return nil
}

func ocSaveKey(work, key string) error {
	lo := exec.Command(ocExe(), "auth", "logout", "zai", ocCredID, "--standalone")
	lo.Dir, lo.Env = work, cleanEnv()
	_ = lo.Run()
	cred := []map[string]any{{
		"id": ocCredID, "integrationID": "zai", "label": "Z.AI", "active": true,
		"value": map[string]string{"type": "key", "key": key},
	}}
	b, _ := json.Marshal(cred)
	rb := make([]byte, 6)
	rand.Read(rb)
	tmp := filepath.Join(os.TempDir(), "aic-"+hex.EncodeToString(rb)+".json")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	defer os.Remove(tmp)
	cmd := exec.Command(ocExe(), "auth", "import", tmp, "--standalone")
	cmd.Dir, cmd.Env = work, cleanEnv()
	out, err := cmd.CombinedOutput()
	if err != nil || !(bytes.Contains(out, []byte("Imported 1")) || bytes.Contains(out, []byte("already present"))) {
		return fmt.Errorf("金鑰存不進 OpenCode：%s", lastLine(stripANSI(string(out))))
	}
	return nil
}

func runTestOC(oc, work string) (string, error) {
	cmd := exec.Command(oc, "run", "--standalone", "請只回覆四個字：準備好了")
	cmd.Dir, cmd.Env = work, cleanEnv()
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	if err := cmd.Start(); err != nil {
		return "", err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		out := stripANSI(buf.String())
		if err != nil || strings.Contains(out, "Error") {
			return "", fmt.Errorf("%s", lastLine(out))
		}
		return lastLine(out), nil
	case <-time.After(150 * time.Second):
		cmd.Process.Kill()
		return "", errors.New("等了 150 秒沒回（網路慢或 Z.AI 太忙）")
	}
}
