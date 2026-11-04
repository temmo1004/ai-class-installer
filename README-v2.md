# AI 課程一鍵安裝 v2（Claude Code ＋ Z.AI GLM-4.5-Flash 免費）

學生只要：**點兩下 → 貼金鑰 → 按 Enter**，之後自動打開 Claude Code（在 `ai-class` 資料夾）。桌面會多一個「開始上課」，下次點它就好。

## 下載
- Windows（一般電腦）：https://github.com/temmo1004/ai-class-installer/releases/download/v2.0.0/AIClass-Windows-x64.exe
- Windows（ARM，Surface Pro X／Copilot+ 筆電）：https://github.com/temmo1004/ai-class-installer/releases/download/v2.0.0/AIClass-Windows-ARM64.exe
- Mac：https://github.com/temmo1004/ai-class-installer/releases/download/v2.0.0/AIClass-Mac.zip

## 學生步驟
1. 下載、點兩下。
   - Windows 跳「Windows 已保護您的電腦」→ 按「其他資訊」→「仍要執行」。
   - Mac：解壓後點兩下「AI課程安裝.command」。跳「無法驗證開發者」→ 按「完成」→ 打開「系統設定 → 隱私權與安全性」，最下面按「仍要打開」→ 輸入密碼。（macOS 14 以前可直接右鍵 →「打開」。）
2. 等它下載（約 120 MB，Windows 另加 Git 60 MB）。
3. 跳出 Z.AI 網頁 → 登入 → 建立 API Key → 複製；貼到跳出的小視窗 → Enter。
4. 自動打開 Claude Code，直接打字問問題。

## 它做了什麼（都不需要管理員）
- 照官方安裝方式裝 Claude Code（`~/.local/bin`，比對 SHA-256）。
- Windows 沒有 Git 就裝 PortableGit 到 `%LOCALAPPDATA%\Programs\ai-class\PortableGit`，設 `CLAUDE_CODE_GIT_BASH_PATH`。
- `~/.claude/settings.json` 的 env：`ANTHROPIC_BASE_URL=https://api.z.ai/api/anthropic`、`ANTHROPIC_AUTH_TOKEN=<金鑰>`、所有模型槽 = `glm-4.5-flash`。
- `~/.claude.json` 設 `hasCompletedOnboarding`、信任 `ai-class`，跳過主題／登入／信任畫面。
- `ai-class/CLAUDE.md`：用繁中回答。
- 再點一次安裝程式＝換金鑰（直接 Enter 不換）。
