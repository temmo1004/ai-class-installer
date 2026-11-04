#!/bin/bash
# AI 課程一鍵安裝（Mac）：在「終端機」貼這一行：
#   curl -fsSL https://github.com/temmo1004/ai-class-installer/releases/latest/download/mac.sh | bash
set -e
f="$(mktemp -d)/aic"
echo "下載安裝程式…"
curl -fsSL -o "$f" https://github.com/temmo1004/ai-class-installer/releases/latest/download/AIClass-Setup-Mac
chmod +x "$f"
exec "$f" < /dev/tty
