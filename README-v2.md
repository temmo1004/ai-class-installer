# AI 課程一鍵安裝 v2.3（OpenCode 或 Claude Code ＋ Z.AI GLM-4.5-Flash 免費）

學生只要：**點兩下 →（選工具，直接 Enter）→ 貼金鑰 → Enter**，之後自動打開（在 `ai-class` 資料夾）。下次點桌面的「開始上課（…）」就好。

## 下載
- Windows（一般電腦）：https://github.com/temmo1004/ai-class-installer/releases/latest/download/AIClass-Setup-Windows-x64.exe
- Windows（ARM，Surface Pro X／Copilot+ 筆電）：https://github.com/temmo1004/ai-class-installer/releases/latest/download/AIClass-Setup-Windows-ARM64.exe
- Mac：https://github.com/temmo1004/ai-class-installer/releases/latest/download/AIClass-Mac.zip

## 兩個工具怎麼選
- 點開會先跳「選要用的工具」：① OpenCode（推薦）、② Claude Code。直接按 Enter＝① OpenCode。
- 已經裝過的會顯示「✓ 已安裝」。兩個都裝過，桌面就有「開始上課（Claude Code）」和「開始上課（OpenCode）」兩個捷徑。
- 金鑰兩邊共用：換金鑰時兩個一起換（只裝過 OpenCode 再改裝 Claude Code 時要重貼一次金鑰）。
- 老師指定今天用哪個（不跳選擇視窗）：
  - 今天用 Claude Code：把 exe 改名成含 `claude`（例如 `AIClass-claude.exe`），或加參數 `--tool=claude`；Mac：`curl -fsSL https://github.com/temmo1004/ai-class-installer/releases/latest/download/mac.sh | bash -s -- --tool=claude`。
  - 檔名含 `opencode` 或 `--tool=opencode` 也可以（跟按 Enter 一樣，只是不跳選擇視窗）。
- Claude Code 在 `ai-class/.claude/settings.json` 預設：改檔不問、常用指令（open/start、ls、find、grep、mkdir、mv、cp、cat、head、tail、wc、curl、unzip、tar）不問；rm 仍然會問。

## 學生步驟
1. 下載、點兩下。
   - Windows 跳「Windows 已保護您的電腦」→ 按「其他資訊」→「仍要執行」。
   - Mac：解壓後點兩下「AI課程安裝.command」。跳「無法驗證開發者」→ 按「完成」→ 打開「系統設定 → 隱私權與安全性」，最下面按「仍要打開」→ 輸入密碼。（macOS 14 以前可直接右鍵 →「打開」。）
2. 選擇畫面直接按 Enter（OpenCode）。等它下載（約 120 MB；選 Claude Code 的 Windows 另加 Git 60 MB）。
3. 跳出 Z.AI 網頁 → 登入 → 建立 API Key → 複製；貼到跳出的小視窗 → Enter。
4. 自動打開，直接打字問問題。

## 它做了什麼（都不需要管理員）
- 照官方安裝方式裝 Claude Code（`~/.local/bin`，比對 SHA-256）。
- Windows 沒有 Git 就裝 PortableGit 到 `%LOCALAPPDATA%\Programs\ai-class\PortableGit`，設 `CLAUDE_CODE_GIT_BASH_PATH`。
- `~/.claude/settings.json` 的 env：`ANTHROPIC_BASE_URL=https://api.z.ai/api/anthropic`、`ANTHROPIC_AUTH_TOKEN=<金鑰>`、所有模型槽 = `glm-4.5-flash`。
- `~/.claude.json` 設 `hasCompletedOnboarding`、信任 `ai-class`，跳過主題／登入／信任畫面。
- `ai-class/CLAUDE.md`：用繁中回答。
- 再點一次安裝程式＝換金鑰（直接 Enter 不換）。
