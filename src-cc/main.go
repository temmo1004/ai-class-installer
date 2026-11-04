// AI 課程一鍵安裝 v2：裝 Claude Code（＋Windows 需要的 Git Bash）→ 填 Z.AI 金鑰 → 接 GLM-4.5-Flash → 自動開。
// 學生只要：點兩下、貼金鑰。不需要系統管理員；再點一次可以換金鑰。
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

const (
	version  = "2.2.0"
	glmModel = "glm-4.5-flash"
	zaiBase  = "https://api.z.ai/api/anthropic"
	keyPage  = "https://z.ai/manage-apikey/apikey-list"
	ccBase   = "https://downloads.claude.ai/claude-code-releases"
	workName = "ai-class"
)

var (
	flagNoStart = flag.Bool("no-start", false, "設定完不打開工具（測試用）")
	flagNoTest  = flag.Bool("no-test", false, "不跑「準備好了」測試")
	flagNoWait  = flag.Bool("no-wait", false, "出錯時不等 Enter（測試用）")
	flagHome    = flag.String("home", "", "改用別的使用者資料夾（測試用）")
	flagTool    = flag.String("tool", "", "直接選工具：claude 或 opencode（不跳選擇視窗）")
)

var stdin = bufio.NewReader(os.Stdin)

func main() {
	flag.Parse()
	platformInit()
	fmt.Printf("\n  ==============================================\n")
	fmt.Printf("    AI 課程一鍵安裝  Claude Code ／ OpenCode（v%s）\n", version)
	fmt.Printf("  ==============================================\n\n")
	if err := run(); err != nil {
		fmt.Printf("\n  ✗ 沒有完成：%v\n", err)
		fmt.Printf("\n  拍下這個畫面舉手，或再點一次這個程式重試。\n")
		pause()
		os.Exit(1)
	}
}

func home() string {
	if *flagHome != "" {
		return *flagHome
	}
	h, _ := os.UserHomeDir()
	return h
}

func pause() {
	if *flagNoWait {
		return
	}
	fmt.Print("\n  按 Enter 關閉 … ")
	stdin.ReadString('\n')
}

// ---------- 安裝 Claude Code（照官方 install.sh／install.ps1：最新版、比對 SHA-256、再跑 claude install） ----------

func installClaude() (string, error) {
	exe := filepath.Join(home(), ".local", "bin", ccName())
	if fi, err := os.Stat(exe); err == nil && fi.Size() > 1<<20 && strings.TrimSpace(output(exe, "", "--version")) != "" {
		fmt.Println("        已經裝過了，略過下載")
		addToPath(filepath.Dir(exe))
		return exe, nil
	}
	ver, err := getText(ccBase + "/latest")
	ver = strings.TrimSpace(ver)
	if err != nil || !regexp.MustCompile(`^\d+\.\d+\.\d+`).MatchString(ver) {
		return "", fmt.Errorf("連不到 Claude Code 下載站（%v）：檢查網路後再點一次", err)
	}
	plat := ccPlatform()
	sumRaw, sizeRaw, _, err := manifest(ver, "manifest.json", plat)
	if err != nil {
		return "", err
	}
	sumZst, _, zName, _ := manifest(ver, "manifest.zst.json", plat)
	tmpDir, _ := os.MkdirTemp("", "aic-cc")
	defer os.RemoveAll(tmpDir)
	bin := filepath.Join(tmpDir, ccName())
	ok := false
	if sumZst != "" && zName != "" {
		z := bin + ".zst"
		if download(fmt.Sprintf("%s/%s/%s/%s", ccBase, ver, plat, zName), z) == nil && fileSum(z) == sumZst {
			if unzstd(z, bin, sizeRaw) == nil && fileSum(bin) == sumRaw {
				ok = true
			}
		}
		os.Remove(z)
	}
	if !ok {
		if err := download(fmt.Sprintf("%s/%s/%s/%s", ccBase, ver, plat, ccName()), bin); err != nil {
			return "", fmt.Errorf("下載失敗（%v）：檢查網路後再點一次", err)
		}
		if fileSum(bin) != sumRaw {
			return "", errors.New("下載的檔案驗證不過（網路不穩？）：再點一次")
		}
	}
	os.Chmod(bin, 0o755)
	afterDownload(bin)
	fmt.Println("        設定 Claude Code 中…")
	cmd := exec.Command(bin, "install")
	cmd.Env = cleanEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("Claude Code 安裝失敗：%s", lastLine(stripANSI(string(out))))
	}
	if strings.TrimSpace(output(exe, "", "--version")) == "" {
		return "", errors.New("裝好了但打不開 claude（防毒軟體擋住？）")
	}
	addToPath(filepath.Dir(exe))
	return exe, nil
}

func manifest(ver, name, plat string) (string, int64, string, error) {
	t, err := getText(fmt.Sprintf("%s/%s/%s", ccBase, ver, name))
	if err != nil {
		return "", 0, "", fmt.Errorf("連不到 Claude Code 下載站（%v）", err)
	}
	var m struct {
		Platforms map[string]struct {
			Checksum string `json:"checksum"`
			Binary   string `json:"binary"`
			Size     int64  `json:"size"`
		} `json:"platforms"`
	}
	if json.Unmarshal([]byte(t), &m) != nil || m.Platforms[plat].Checksum == "" {
		return "", 0, "", fmt.Errorf("Claude Code 沒有 %s 版", plat)
	}
	p := m.Platforms[plat]
	if p.Binary == "" && name == "manifest.zst.json" {
		p.Binary = "claude.zst"
	}
	return p.Checksum, p.Size, p.Binary, nil
}

func unzstd(src, dst string, size int64) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	d, err := zstd.NewReader(f)
	if err != nil {
		return err
	}
	defer d.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	var r io.Reader = d
	if size > 0 {
		r = io.LimitReader(d, size)
	}
	_, err = io.Copy(out, r)
	return err
}

func fileSum(p string) string {
	f, err := os.Open(p)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil))
}

func getText(url string) (string, error) {
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return string(b), err
}

func download(url, dst string) error {
	var last error
	for try := 1; try <= 3; try++ {
		if try > 1 {
			fmt.Printf("        重試第 %d 次…\n", try)
			time.Sleep(3 * time.Second)
		}
		if last = downloadOnce(url, dst); last == nil {
			return nil
		}
	}
	return last
}

func downloadOnce(url, dst string) error {
	resp, err := (&http.Client{Timeout: 30 * time.Minute}).Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	total, got := resp.ContentLength, int64(0)
	buf, last := make([]byte, 256<<10), time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			got += int64(n)
			if time.Since(last) > 700*time.Millisecond {
				last = time.Now()
				if total > 0 {
					fmt.Printf("\r        %3d%%  %.0f / %.0f MB   ", got*100/total, float64(got)/1e6, float64(total)/1e6)
				} else {
					fmt.Printf("\r        %.0f MB   ", float64(got)/1e6)
				}
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			fmt.Println()
			return rerr
		}
	}
	fmt.Printf("\r        100%%  %.0f MB                 \n", float64(got)/1e6)
	if total > 0 && got != total {
		return fmt.Errorf("只下載到 %d / %d", got, total)
	}
	return nil
}

// ---------- 金鑰 ----------

func askKey(hadKey bool) (string, error) {
	if f := os.Getenv("AIC_KEYFILE"); f != "" { // 測試用：不跳視窗、不放命令列
		b, _ := os.ReadFile(f)
		k := cleanKey(string(b))
		if msg := checkKey(k); msg != "" {
			return "", errors.New(msg)
		}
		return k, nil
	}
	if !hadKey {
		fmt.Println("        正在打開 Z.AI 金鑰網頁：登入 → 建立金鑰（Create API Key）→ 按複製")
		openURL(keyPage)
	}
	hint := "到剛打開的 Z.AI 網頁建立金鑰、複製，貼在下面，按 Enter。"
	if hadKey {
		hint = "已經有金鑰了。要換就貼新的金鑰；不換就直接按 Enter。"
	}
	for i := 0; i < 20; i++ {
		raw, ok := keyDialog(hint)
		if !ok {
			fmt.Printf("        %s\n        金鑰：", hint)
			raw, _ = stdin.ReadString('\n')
		}
		k := cleanKey(raw)
		if k == "" && hadKey {
			return "", nil
		}
		if k == "" {
			hint = "還沒貼金鑰喔。到 Z.AI 網頁複製金鑰，貼在下面，按 Enter。"
			continue
		}
		if msg := checkKey(k); msg != "" {
			fmt.Printf("        ✗ %s\n", msg)
			hint = msg + "請重新複製一次，貼在下面。"
			continue
		}
		return k, nil
	}
	return "", errors.New("金鑰一直沒填好")
}

func cleanKey(s string) string {
	s = strings.Trim(strings.TrimSpace(s), "\"'“”‘’`")
	return strings.Join(strings.Fields(s), "")
}

// 真的打一次 Z.AI 的 Anthropic 相容端點（Claude Code 用的就是這個）
func checkKey(k string) string {
	if len(k) < 20 {
		return "金鑰太短（少複製到一段？）。"
	}
	for _, r := range k {
		if r > 126 || r < 33 {
			return "金鑰裡有中文或奇怪的字（複製到別的東西？）。"
		}
	}
	fmt.Println("        檢查金鑰能不能用…")
	body := `{"model":"` + glmModel + `","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`
	for try := 0; try < 3; try++ {
		req, _ := http.NewRequest("POST", zaiBase+"/v1/messages", strings.NewReader(body))
		req.Header.Set("x-api-key", k)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("content-type", "application/json")
		resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
		if err != nil {
			fmt.Printf("        ! 連不到 Z.AI 檢查（%s），先存起來\n", shortErr(err))
			return ""
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		switch {
		case resp.StatusCode == 200:
			fmt.Println("        ✓ 金鑰可以用")
			return ""
		case resp.StatusCode == 401 || resp.StatusCode == 403:
			return "這把金鑰 Z.AI 不認得（複製錯、或已經刪掉）。"
		case bytes.Contains(b, []byte("1305")) || resp.StatusCode == 529:
			time.Sleep(3 * time.Second) // 太忙，再試
			continue
		default:
			fmt.Printf("        ! Z.AI 回 %d，先存起來\n", resp.StatusCode)
			return ""
		}
	}
	fmt.Println("        ! Z.AI 現在很忙，先存起來")
	return ""
}

func shortErr(err error) string {
	s := err.Error()
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

// ---------- 設定 ----------

func readJSON(p string) map[string]any {
	m := map[string]any{}
	if b, err := os.ReadFile(p); err == nil && len(bytes.TrimSpace(b)) > 0 {
		if json.Unmarshal(b, &m) != nil {
			os.Rename(p, p+".bak")
			m = map[string]any{}
		}
	}
	return m
}

func writeJSON(p string, m map[string]any, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(p, append(b, '\n'), mode)
}

func currentKey(settings string) string {
	if env, ok := readJSON(settings)["env"].(map[string]any); ok {
		if s, _ := env["ANTHROPIC_AUTH_TOKEN"].(string); s != "" && env["ANTHROPIC_BASE_URL"] == zaiBase {
			return s
		}
	}
	return ""
}

func writeSettings(p, key string, extra map[string]string) error {
	m := readJSON(p)
	env, _ := m["env"].(map[string]any)
	if env == nil {
		env = map[string]any{}
	}
	for k, v := range map[string]string{
		"ANTHROPIC_BASE_URL":                       zaiBase,
		"ANTHROPIC_AUTH_TOKEN":                     key,
		"ANTHROPIC_MODEL":                          glmModel,
		"ANTHROPIC_DEFAULT_OPUS_MODEL":             glmModel,
		"ANTHROPIC_DEFAULT_SONNET_MODEL":           glmModel,
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":            glmModel,
		"CLAUDE_CODE_SUBAGENT_MODEL":               glmModel,
		"API_TIMEOUT_MS":                           "3000000",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
		"CLAUDE_CODE_AUTO_COMPACT_WINDOW":          "128000",
	} {
		env[k] = v
	}
	for k, v := range extra {
		env[k] = v
	}
	delete(env, "ANTHROPIC_API_KEY")
	m["env"] = env
	return writeJSON(p, m, 0o600)
}

// 跳過第一次打開的主題選擇、登入畫面、「信任這個資料夾嗎」
// 專案層權限（只在 ai-class 生效）：改檔不問、常用無害指令不問；不給 rm、不開 bypass
var allowRules = []string{
	"Bash(open *)", "Bash(start *)", "Bash(cmd //c start *)", "Bash(cmd.exe /c start *)", "Bash(explorer *)", "Bash(explorer.exe *)",
	"Bash(ls)", "Bash(ls *)", "Bash(dir)", "Bash(dir *)", "Bash(pwd)", "Bash(find *)", "Bash(mkdir *)",
	"Bash(mv *)", "Bash(cp *)", "Bash(cat *)", "Bash(head *)", "Bash(tail *)", "Bash(wc *)",
	"Bash(curl *)", "Bash(unzip *)", "Bash(tar *)",
	"Bash(powershell Expand-Archive *)", "Bash(powershell -Command Expand-Archive *)",
	"WebFetch",
}

func projectPerms(work string) error {
	p := filepath.Join(work, ".claude", "settings.json")
	m := readJSON(p)
	perm, _ := m["permissions"].(map[string]any)
	if perm == nil {
		perm = map[string]any{}
	}
	perm["defaultMode"] = "acceptEdits"
	have := map[string]bool{}
	var allow []any
	if old, ok := perm["allow"].([]any); ok {
		for _, r := range old {
			if s, _ := r.(string); s != "" && !have[s] {
				have[s] = true
				allow = append(allow, s)
			}
		}
	}
	for _, r := range allowRules {
		if !have[r] {
			allow = append(allow, r)
		}
	}
	perm["allow"] = allow
	m["permissions"] = perm
	return writeJSON(p, m, 0o644)
}

func skipOnboarding(work string) error {
	if err := projectPerms(work); err != nil {
		return err
	}
	p := filepath.Join(home(), ".claude.json")
	m := readJSON(p)
	m["hasCompletedOnboarding"] = true
	if _, ok := m["theme"]; !ok {
		m["theme"] = "dark"
	}
	projects, _ := m["projects"].(map[string]any)
	if projects == nil {
		projects = map[string]any{}
	}
	for _, w := range []string{work, filepath.ToSlash(work)} {
		pr, _ := projects[w].(map[string]any)
		if pr == nil {
			pr = map[string]any{}
		}
		pr["hasTrustDialogAccepted"] = true
		pr["hasCompletedProjectOnboarding"] = true
		projects[w] = pr
	}
	m["projects"] = projects
	return writeJSON(p, m, 0o600)
}

func runTestCC(cc, work string) (string, error) {
	cmd := exec.Command(cc, "-p", "請只回覆四個字：準備好了")
	cmd.Dir = work
	cmd.Env = cleanEnv()
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
		if err != nil {
			return "", fmt.Errorf("%s", lastLine(out))
		}
		return lastLine(out), nil
	case <-time.After(180 * time.Second):
		cmd.Process.Kill()
		return "", errors.New("等了 180 秒沒回（網路慢或 Z.AI 太忙）")
	}
}

// 不讓別的設定蓋掉：拿掉 ANTHROPIC_*／CLAUDE*（設定都放在 settings.json）
func cleanEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		u := strings.ToUpper(e)
		if strings.HasPrefix(u, "ANTHROPIC_") || strings.HasPrefix(u, "CLAUDE") || strings.HasPrefix(u, "ZAI_API_KEY=") || strings.HasPrefix(u, "ZHIPU_API_KEY=") {
			continue
		}
		if *flagHome != "" && (strings.HasPrefix(u, "HOME=") || strings.HasPrefix(u, "USERPROFILE=") || strings.HasPrefix(u, "LOCALAPPDATA=") || strings.HasPrefix(u, "APPDATA=") || strings.HasPrefix(u, "XDG_")) {
			continue
		}
		env = append(env, e)
	}
	if *flagHome != "" {
		env = append(env, "HOME="+*flagHome, "USERPROFILE="+*flagHome,
			"LOCALAPPDATA="+filepath.Join(*flagHome, "AppData", "Local"),
			"APPDATA="+filepath.Join(*flagHome, "AppData", "Roaming"))
	}
	return env
}

func output(name, dir string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = cleanEnv()
	b, _ := cmd.Output()
	return stripANSI(string(b))
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && !(s[j] >= '@' && s[j] <= '~') {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func lastLine(s string) string {
	ls := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(ls) - 1; i >= 0; i-- {
		if t := strings.TrimSpace(ls[i]); t != "" {
			if len(t) > 160 {
				t = t[:160]
			}
			return t
		}
	}
	return "（沒有輸出）"
}
