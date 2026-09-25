#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BOOT_DIR="$HOME/.termux/boot"
BOOT_SCRIPT="$BOOT_DIR/start-super-picos"

if [[ ! -x "$APP_DIR/super-picos" ]]; then
  echo "Erro: binário super-picos não encontrado em $APP_DIR" >&2
  exit 1
fi

if [[ ! -f "$APP_DIR/.env" ]]; then
  echo "Erro: crie $APP_DIR/.env antes de instalar o autostart." >&2
  exit 1
fi

mkdir -p "$BOOT_DIR"

cat > "$BOOT_SCRIPT" <<EOF_BOOT
#!/data/data/com.termux/files/usr/bin/sh

termux-wake-lock 2>/dev/null || true
export HOME=$HOME
export PATH=/data/data/com.termux/files/usr/bin:\$PATH

cd "$APP_DIR" || exit 1

if pgrep -f "$APP_DIR/super-picos" >/dev/null 2>&1; then
  exit 0
fi

exec "$APP_DIR/super-picos" >> "$APP_DIR/bot.log" 2>&1
EOF_BOOT

chmod +x "$BOOT_SCRIPT"

echo "Script de boot instalado em: $BOOT_SCRIPT"
echo "Abra o Termux:Boot ao menos uma vez e deixe o Termux sem restrição de bateria."
echo "Logs: tail -f '$APP_DIR/bot.log'"
