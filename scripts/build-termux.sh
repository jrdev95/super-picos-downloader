#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARCH="$(go env GOARCH)"
OUT="$ROOT/dist/termux-$ARCH"
ARCHIVE="$ROOT/dist/super-picos-termux-$ARCH.tar.gz"

rm -rf "$OUT"
mkdir -p "$OUT/secrets"

cd "$ROOT"

echo "==> Executando testes"
go test ./...
go vet ./...

echo "==> Compilando nativamente para Termux ($ARCH)"
go build -trimpath -ldflags="-s -w" \
  -o "$OUT/super-picos" ./cmd/bot

cp .env.example README.md "$OUT/"
cp packaging/termux/install-autostart.sh "$OUT/"
chmod +x "$OUT/super-picos" "$OUT/install-autostart.sh"

cat > "$OUT/secrets/README.txt" <<'TXT'
Coloque aqui o arquivo instagram-cookies.txt, se necessário.
Nunca distribua cookies reais dentro de um release.
TXT

rm -f "$ARCHIVE"
tar -czf "$ARCHIVE" -C "$OUT" .

echo "==> Release criado: $ARCHIVE"
