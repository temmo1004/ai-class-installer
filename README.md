# AI 課程一鍵安裝（OpenCode ＋ GLM）

點一下就裝好 OpenCode，填入 Z.AI 金鑰，預設用 GLM-4.5-Flash（免費）。不需要系統管理員權限。

## Windows

1. 下載 [AIClass-Setup-Windows-x64.exe](https://github.com/temmo1004/ai-class-installer/releases/latest/download/AIClass-Setup-Windows-x64.exe)（ARM 筆電也能用；或下載 [ARM64 版](https://github.com/temmo1004/ai-class-installer/releases/latest/download/AIClass-Setup-Windows-ARM64.exe)）
2. 點兩下。跳出藍色「Windows 已保護您的電腦」→ 按「其他資訊」→「仍要執行」
3. 等它裝好，跳出視窗 → 貼上 Z.AI 金鑰 → 按 Enter
4. 再按一次 Enter，OpenCode 就開好了

之後要再開：點桌面的「開始上課 OpenCode.cmd」。要換金鑰：再點一次安裝程式。

## Mac

打開「終端機」，貼這一行，按 Enter：

```
curl -fsSL https://github.com/temmo1004/ai-class-installer/releases/latest/download/mac.sh | bash
```

跳出視窗 → 貼上 Z.AI 金鑰 → 按 Enter → 再按一次 Enter。之後要再開：點桌面的「開始上課 OpenCode.command」。

## 金鑰

到 https://z.ai/manage-apikey/apikey-list 登入 → Create API Key → 複製。

## 它做了什麼

- 從 opencode.ai 官方下載 OpenCode（Windows 放 `%LOCALAPPDATA%\Programs\ai-class`，Mac 放 `~/.opencode/bin`），加進使用者 PATH
- 建工作資料夾 `~/ai-class`
- 先打一次 Z.AI 確認金鑰能用，再用 `opencode auth import` 存進 OpenCode 自己的資料（不會留在任何文字檔）
- OpenCode 設定檔加上 `"model": "zai/glm-4.5-flash"`
- 請 AI 回一句「準備好了」確認全部通了
