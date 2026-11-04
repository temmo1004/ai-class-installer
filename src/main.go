// AI 課程一鍵安裝：裝 OpenCode → 填 Z.AI 金鑰 → 設好 GLM-4.5-Flash → 直接開始。
// 不需要系統管理員；再跑一次可以換金鑰。
package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/rand"
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
	"runtime"
	"strings"
	"time"
)

const (
	version   = "1.0.0"
	model     = "zai/glm-4.5-flash"
	keyPage   = "https://z.ai/manage-apikey/apikey-list"
	credID    = "cred_aiclassinstaller0000000001"
	dlBase    = "https://opencode.ai/update/api/latest/cli/opencode/opencode-"
	zaiChat   = "https://api.z.ai/api/paas/v4/chat/completions"
	workName  = "ai-class"
	launcherW = "開始上課 OpenCode.cmd"
	launcherM = "開始上課 OpenCode.command"
)

var (
	flagKey     = flag.String("key", "", "直接給金鑰（不跳視窗，測試用）")
	flagNoStart = flag.Bool("no-start", false, "設定完不開 OpenCode（測試用）")
	flagNoTest  = flag.Bool("no-test", false, "不跑「準備好了」測試")
	flagNoWait  = flag.Bool("no-wait", false, "結束時不等 Enter（測試用）")
	flagHome    = flag.String("home", "", "改用別的使用者資料夾（測試用）")
)

var stdin = bufio.NewReader(os.Stdin)

func main() {
	flag.Parse()
	platformInit()
	fmt.Printf("\n  ==============================================\n")
	fmt.Printf("    AI 課程一鍵安裝  OpenCode ＋ GLM（v%s）\n", version)
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

func run() error {
	// ① 安裝 OpenCode
	fmt.Println("  【1/4】安裝 OpenCode")
	oc, err := installOpenCode()
	if err != nil {
		return err
	}
	ver := strings.TrimSpace(output(oc, "", "--version"))
	fmt.Printf("        ✓ 裝好了（版本 %s）\n        位置：%s\n\n", ver, oc)

	// ② 工作資料夾
	work := filepath.Join(home(), workName)
	if err := os.MkdirAll(work, 0o755); err != nil {
		return fmt.Errorf("建不了資料夾 %s：%v", work, err)
	}
	fmt.Printf("  【2/4】工作資料夾\n        ✓ %s\n\n", work)

	// ③ 金鑰
	fmt.Println("  【3/4】填 Z.AI 金鑰")
	cfgDir := strings.TrimSpace(output(oc, work, "debug", "paths", "config"))
	if cfgDir == "" {
		return errors.New("找不到 OpenCode 的設定資料夾")
	}
	marker := filepath.Join(cfgDir, "ai-class-installer.json")
	_, had := os.Stat(marker)
	hadKey := had == nil
	key, err := askKey(hadKey)
	if err != nil {
		return err
	}
	if key == "" {
		fmt.Println("        ✓ 沿用原本的金鑰")
	} else {
		if err := saveKey(oc, work, key); err != nil {
			return err
		}
		fmt.Println("        ✓ 金鑰存好了（存在 OpenCode 自己的資料裡，不會出現在檔案總管）")
	}
	if err := setModel(cfgDir); err != nil {
		return err
	}
	_ = os.WriteFile(marker, []byte(`{"installer":"`+version+`","model":"`+model+`","at":"`+time.Now().Format(time.RFC3339)+`"}`+"\n"), 0o644)
	fmt.Printf("        ✓ 預設模型：%s\n\n", model)

	// ④ 測一下
	fmt.Println("  【4/4】測試：請 AI 回一句話（約 10–40 秒）")
	if !*flagNoTest {
		t0 := time.Now()
		out, err := runTest(oc, work)
		if err != nil {
			fmt.Printf("        ! 測試沒過：%v\n        先繼續，打開 OpenCode 後問它一句話試試。\n\n", err)
		} else {
			fmt.Printf("        ✓ AI 回：%s（%.0f 秒）\n\n", out, time.Since(t0).Seconds())
		}
	} else {
		fmt.Println("        （略過）")
	}

	launcher := makeLauncher(oc, work)
	fmt.Println("  ----------------------------------------------")
	fmt.Println("  全部好了！")
	if launcher != "" {
		fmt.Printf("  之後要再開：點桌面的「%s」\n", filepath.Base(launcher))
	}
	fmt.Println("  要換金鑰：再點一次這個安裝程式")
	fmt.Println("  ----------------------------------------------")
	if *flagNoStart {
		return nil
	}
	fmt.Print("\n  按 Enter 開始使用 OpenCode … ")
	if !*flagNoWait {
		stdin.ReadString('\n')
	}
	cmd := exec.Command(oc)
	cmd.Dir = work
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = cleanEnv()
	return cmd.Run()
}

func pause() {
	if *flagNoWait {
		return
	}
	fmt.Print("\n  按 Enter 關閉 … ")
	stdin.ReadString('\n')
}

// ---------- 安裝 ----------

func installOpenCode() (string, error) {
	dir := binDir()
	exe := filepath.Join(dir, ocName())
	if fi, err := os.Stat(exe); err == nil && fi.Size() > 1<<20 {
		if v := strings.TrimSpace(output(exe, "", "--version")); v != "" {
			fmt.Println("        已經裝過了，略過下載")
			addToPath(dir)
			return exe, nil
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("建不了 %s：%v", dir, err)
	}
	url := dlBase + platformName() + ".zip"
	fmt.Printf("        下載中（約 50–110 MB，網路慢要幾分鐘，請不要關掉這個視窗）\n")
	tmp := filepath.Join(os.TempDir(), "ai-class-opencode.zip")
	if err := download(url, tmp); err != nil {
		return "", fmt.Errorf("下載失敗（%v）：檢查網路後再點一次", err)
	}
	defer os.Remove(tmp)
	if err := unzipOne(tmp, ocName(), exe); err != nil {
		return "", fmt.Errorf("解壓縮失敗：%v", err)
	}
	afterInstall(exe)
	if strings.TrimSpace(output(exe, "", "--version")) == "" {
		return "", errors.New("裝好了但打不開 opencode（防毒軟體擋住？）")
	}
	addToPath(dir)
	return exe, nil
}

func download(url, dst string) error {
	var last error
	for try := 1; try <= 3; try++ {
		if try > 1 {
			fmt.Printf("        重試第 %d 次…\n", try)
			time.Sleep(3 * time.Second)
		}
		last = downloadOnce(url, dst)
		if last == nil {
			return nil
		}
	}
	return last
}

func downloadOnce(url, dst string) error {
	c := &http.Client{Timeout: 20 * time.Minute}
	resp, err := c.Get(url)
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
	total := resp.ContentLength
	var got int64
	buf := make([]byte, 256<<10)
	last := time.Now()
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
					fmt.Printf("\r        %3d%%  %.1f / %.1f MB   ", got*100/total, float64(got)/1e6, float64(total)/1e6)
				} else {
					fmt.Printf("\r        %.1f MB   ", float64(got)/1e6)
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
	fmt.Printf("\r        100%%  %.1f MB                 \n", float64(got)/1e6)
	if total > 0 && got != total {
		return fmt.Errorf("只下載到 %d / %d", got, total)
	}
	return nil
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

// ---------- 金鑰 ----------

func askKey(hadKey bool) (string, error) {
	if *flagKey == "" && os.Getenv("AIC_KEYFILE") != "" { // 測試用：從檔案讀金鑰，不放在命令列
		b, _ := os.ReadFile(os.Getenv("AIC_KEYFILE"))
		*flagKey = string(b)
	}
	if *flagKey != "" {
		k := cleanKey(*flagKey)
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
		if !ok { // 視窗開不了 → 用這個黑視窗問
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
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"'“”‘’`")
	return strings.Join(strings.Fields(s), "")
}

// checkKey：格式＋真的打一次 Z.AI。回空字串＝可以用。
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
	body := `{"model":"glm-4.5-flash","messages":[{"role":"user","content":"hi"}],"max_tokens":8}`
	req, _ := http.NewRequest("POST", zaiChat, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+k)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 40 * time.Second}).Do(req)
	if err != nil {
		fmt.Printf("        ! 連不到 Z.AI 檢查（%v），先存起來\n", shortErr(err))
		return ""
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	switch {
	case resp.StatusCode == 200:
		fmt.Println("        ✓ 金鑰可以用")
		return ""
	case resp.StatusCode == 401 || resp.StatusCode == 403:
		return "這把金鑰 Z.AI 不認得（複製錯、或已經刪掉）。"
	default:
		fmt.Printf("        ! Z.AI 回 %d（可能太忙），先存起來\n", resp.StatusCode)
		return ""
	}
}

func shortErr(err error) string {
	s := err.Error()
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

func saveKey(oc, work, key string) error {
	// 換金鑰：先拿掉這支程式上次存的那一把（沒有也沒關係），再匯入新的
	lo := exec.Command(oc, "auth", "logout", "zai", credID, "--standalone")
	lo.Dir, lo.Env = work, cleanEnv()
	_ = lo.Run()
	cred := []map[string]any{{
		"id": credID, "integrationID": "zai", "label": "Z.AI", "active": true,
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
	cmd := exec.Command(oc, "auth", "import", tmp, "--standalone")
	cmd.Dir = work
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	if err != nil || !(bytes.Contains(out, []byte("Imported 1")) || bytes.Contains(out, []byte("already present"))) {
		return fmt.Errorf("金鑰存不進 OpenCode：%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func setModel(cfgDir string) error {
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return err
	}
	p := filepath.Join(cfgDir, "opencode.json")
	cfg := map[string]any{}
	if b, err := os.ReadFile(p); err == nil && len(bytes.TrimSpace(b)) > 0 {
		if json.Unmarshal(b, &cfg) != nil {
			os.Rename(p, p+".bak")
			cfg = map[string]any{}
		}
	}
	if _, ok := cfg["$schema"]; !ok {
		cfg["$schema"] = "https://opencode.ai/config.json"
	}
	cfg["model"] = model
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(p, append(b, '\n'), 0o644)
}

func runTest(oc, work string) (string, error) {
	cmd := exec.Command(oc, "run", "--standalone", "請只回覆四個字：準備好了")
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
		if err != nil || strings.Contains(out, "Error") {
			return "", fmt.Errorf("%s", lastLine(out))
		}
		return lastLine(out), nil
	case <-time.After(150 * time.Second):
		cmd.Process.Kill()
		return "", errors.New("等了 150 秒沒回（網路慢或 Z.AI 太忙）")
	}
}

// 不讓舊的環境變數蓋掉剛存的金鑰
func cleanEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		u := strings.ToUpper(e)
		if strings.HasPrefix(u, "ZAI_API_KEY=") || strings.HasPrefix(u, "ZHIPU_API_KEY=") {
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

var _ = runtime.GOOS
