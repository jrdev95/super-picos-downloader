#!/usr/bin/env bash
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE_FILE="$SERVICE_DIR/super-picos.service"

if [[ ! -x "$APP_DIR/super-picos" ]]; then
  echo "Erro: binário super-picos não encontrado em $APP_DIR" >&2
  exit 1
fi

if [[ ! -f "$APP_DIR/.env" ]]; then
  echo "Erro: crie $APP_DIR/.env antes de instalar o autostart." >&2
  exit 1
fi

mkdir -p "$SERVICE_DIR"

cat > "$SERVICE_FILE" <<EOF_SERVICE
[Unit]
Description=Super Picos Downloader

[Service]
Type=simple
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/super-picos
Environment=HOME=$HOME
Environment=PATH=$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
EOF_SERVICE

systemctl --user daemon-reload
systemctl --user enable --now super-picos.service

echo "Serviço instalado e iniciado."
echo "Status: systemctl --user status super-picos.service"
echo "Logs:   journalctl --user -u super-picos.service -f"

if command -v loginctl >/dev/null 2>&1; then
  if [[ "$(loginctl show-user "$USER" -p Linger --value 2>/dev/null || true)" != "yes" ]]; then
    echo
    echo "Para iniciar no boot antes do login, execute:"
    echo "  sudo loginctl enable-linger $USER"
  fi
fi
