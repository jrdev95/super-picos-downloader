# Super Picos Downloader

Bot do Telegram escrito em Go para baixar mídias de links enviados no chat e responder com a mídia correspondente.

O projeto prioriza comportamento previsível, limpeza automática dos arquivos temporários, múltiplas estratégias de download e execução multiplataforma.

## Status

A aplicação foi validada em três ambientes:

- Linux `amd64`
- Windows `amd64`
- Android `arm64` via Termux

Também foram validados os respectivos métodos de inicialização automática.

## Plataformas suportadas

| Plataforma | Suporte atual | Estratégia principal |
| --- | --- | --- |
| TikTok | Vídeo e carrossel de fotos | `yt-dlp` + `gallery-dl` |
| Instagram | Foto, vídeo, Reel e carrossel | `gallery-dl` + cookies, com fallback `yt-dlp` |
| Threads | Foto, vídeo e carrossel | Downloader próprio em Go |
| X / Twitter | Foto, vídeo e múltiplas mídias | `gallery-dl` + fallback `yt-dlp` |
| YouTube Shorts | Vídeo | `yt-dlp` |
| Reddit | Vídeo e foto única | `yt-dlp` + fallback RSS |
| Erome | Foto e vídeo | Downloader próprio em Go |

### Limitações conhecidas

- Galerias do Reddit ainda não são suportadas de forma confiável. O RSS público não fornece todas as imagens e o endpoint JSON pode bloquear acesso anônimo.
- A Bot API oficial do Telegram limita uploads multipart a 10 MB para fotos e 50 MB para outros arquivos, incluindo vídeos. Arquivos acima desses limites podem falhar no envio.
- Um Telegram Bot API Server próprio, em modo local, permite uploads de até 2000 MB. Esse modo não faz parte da instalação padrão deste projeto por enquanto.

Referências oficiais:

- Telegram Bot API: <https://core.telegram.org/bots/api>
- Telegram Bot API Server: <https://github.com/tdlib/telegram-bot-api>

## Comportamento do bot

- Processa somente links recebidos enquanto o bot está rodando.
- Descarta updates pendentes existentes antes de iniciar o polling normal.
- Responde à mensagem original no Telegram.
- Exibe uma mensagem temporária durante o download.
- Envia 2 ou mais mídias como álbum.
- Divide álbuns com mais de 10 itens sem criar um grupo contendo apenas 1 mídia.
- Usa fila de jobs e múltiplos workers.
- Cada download possui timeout de 3 minutos.
- Remove o workspace temporário assim que o processamento termina, inclusive em caso de erro.
- Responde a interrupção de terminal e `SIGTERM`, permitindo encerramento mais limpo em Linux/Termux.

## Requisitos

### Aplicação

- Go 1.22 ou superior para compilar a partir do código-fonte.

### Ferramentas externas em tempo de execução

- `yt-dlp`
- `ffmpeg`
- `gallery-dl`

As ferramentas precisam estar disponíveis no `PATH` do processo que executa o bot.

## Configuração

Copie o exemplo:

```bash
cp .env.example .env
```

No Windows PowerShell:

```powershell
Copy-Item .env.example .env
```

Configure:

```env
BOT_TOKEN=SEU_TOKEN_DO_TELEGRAM
MAX_WORKERS=3
INSTAGRAM_COOKIES_FILE=secrets/instagram-cookies.txt
```

### `BOT_TOKEN`

Token do bot criado no BotFather.

### `MAX_WORKERS`

Quantidade máxima de downloads processados simultaneamente. O padrão é `3`.

### `INSTAGRAM_COOKIES_FILE`

Opcional, mas muitos conteúdos do Instagram exigem sessão autenticada.

O arquivo deve usar o formato Netscape/Mozilla `cookies.txt`. A localização recomendada é:

```text
secrets/instagram-cookies.txt
```

A pasta `secrets/`, `.env` e arquivos de cookies são ignorados pelo Git.

**Nunca publique o token do bot nem cookies de sessão.**

---

# Instalação por plataforma

## Linux

### Dependências

Em Ubuntu, Debian, KDE neon e derivados:

```bash
sudo apt update
sudo apt install -y ffmpeg python3 python3-pip pipx
pipx ensurepath
pipx install "yt-dlp[default,curl-cffi]"
pipx install gallery-dl
```

Instale Go 1.22+ pelo método apropriado da distribuição.

Valide:

```bash
go version
yt-dlp --version
gallery-dl --version
ffmpeg -version
```

### Rodar pelo código-fonte

```bash
go mod download
go run ./cmd/bot
```

### Compilar

```bash
mkdir -p bin
go build -trimpath -ldflags="-s -w" -o bin/super-picos ./cmd/bot
```

### Gerar pacote Linux `amd64`

```bash
./scripts/build-linux.sh
```

O pacote será criado em:

```text
dist/super-picos-linux-amd64.tar.gz
```

### Autostart no Linux

Depois de extrair o pacote, criar o `.env` e adicionar os cookies se necessário:

```bash
./install-autostart.sh
```

O instalador cria um serviço `systemd --user` chamado:

```text
super-picos.service
```

Status:

```bash
systemctl --user status super-picos.service
```

Logs:

```bash
journalctl --user -u super-picos.service -f
```

Para iniciar no boot mesmo antes do login:

```bash
sudo loginctl enable-linger "$USER"
```

---

## Windows

### Dependências

No PowerShell, Go e FFmpeg podem ser instalados com `winget`:

```powershell
winget install -e --id GoLang.Go
winget install -e --id Gyan.FFmpeg
winget install -e --id Python.Python.3.13
```

Feche e reabra o terminal após instalações que alterem o `PATH`.

Instale `yt-dlp` e `gallery-dl` para o usuário atual:

```powershell
py -m pip install --user --upgrade "yt-dlp[default,curl-cffi]" gallery-dl
```

Descubra o diretório correto de scripts do Python:

```powershell
$Scripts = py -c "import sysconfig; print(sysconfig.get_path('scripts', scheme='nt_user'))"
$Scripts
```

Caso esse diretório ainda não esteja no `PATH`, adicione-o permanentemente:

```powershell
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")

if (($userPath -split ';') -notcontains $Scripts) {
    [Environment]::SetEnvironmentVariable(
        "Path",
        $userPath.TrimEnd(';') + ";" + $Scripts,
        "User"
    )
}
```

Feche e reabra o PowerShell e valide:

```powershell
go version
yt-dlp --version
gallery-dl --version
ffmpeg -version
```

### Rodar pelo código-fonte

```powershell
go mod download
go run .\cmd\bot
```

### Gerar pacote Windows `amd64`

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1
```

O pacote será criado em:

```text
dist\super-picos-windows-amd64.zip
```

Ele contém dois executáveis:

```text
super-picos.exe
super-picos-hidden.exe
```

- `super-picos.exe`: versão de console para diagnóstico e logs.
- `super-picos-hidden.exe`: versão sem janela de console, destinada ao autostart.

### Autostart no Windows

Depois de extrair o pacote, criar `.env` e configurar `secrets/`:

```powershell
powershell -ExecutionPolicy Bypass -File .\register-autostart.ps1
```

O script registra a tarefa:

```text
Super Picos Downloader
```

Diagnóstico:

```powershell
Get-ScheduledTaskInfo -TaskName "Super Picos Downloader" |
Format-List LastRunTime,LastTaskResult
```

Verificar o processo oculto:

```powershell
Get-Process super-picos-hidden
```

Parar:

```powershell
Stop-ScheduledTask -TaskName "Super Picos Downloader"
Stop-Process -Name super-picos-hidden -Force
```

Remover o autostart:

```powershell
Unregister-ScheduledTask -TaskName "Super Picos Downloader" -Confirm:$false
```

---

## Android / Termux

Mantenha o projeto e o executável dentro do `$HOME` do Termux. Evite executar o binário diretamente em `/sdcard` ou `~/storage/shared`.

### Dependências

```bash
pkg update && pkg upgrade -y
pkg install -y golang python ffmpeg git python-yt-dlp
python -m pip install --upgrade gallery-dl
```

Valide:

```bash
go version
python --version
yt-dlp --version
gallery-dl --version
ffmpeg -version
```

### Rodar pelo código-fonte

```bash
go run ./cmd/bot
```

### Compilar nativamente

```bash
mkdir -p bin
go build -trimpath -ldflags="-s -w" -o bin/super-picos ./cmd/bot
```

### Gerar pacote Termux

```bash
./scripts/build-termux.sh
```

A arquitetura é detectada automaticamente pelo Go. Em um aparelho ARM64, o resultado será semelhante a:

```text
dist/super-picos-termux-arm64.tar.gz
```

### Autostart com Termux:Boot

Instale o Termux:Boot da mesma origem/assinatura do Termux e abra o aplicativo pelo menos uma vez.

Depois de extrair o pacote, criar `.env` e configurar `secrets/`:

```bash
./install-autostart.sh
```

O instalador cria:

```text
~/.termux/boot/start-super-picos
```

O processo usa `termux-wake-lock` quando disponível e grava logs em:

```text
bot.log
```

Acompanhar:

```bash
tail -f bot.log
```

No Android, deixe o Termux e o Termux:Boot sem restrição de bateria para reduzir a chance de o sistema encerrar o processo em segundo plano.

---

# Releases locais

Os scripts de build criam pacotes dentro de `dist/` e **não copiam** `.env` nem cookies reais.

Estrutura típica:

```text
dist/
├── linux-amd64/
├── windows-amd64/
├── termux-arm64/
├── super-picos-linux-amd64.tar.gz
├── super-picos-windows-amd64.zip
└── super-picos-termux-arm64.tar.gz
```

Os pacotes incluem:

```text
.env.example
README.md
secrets/README.txt
```

O usuário deve criar o próprio `.env` e adicionar o próprio `instagram-cookies.txt` após extrair o pacote.

## Scripts disponíveis

```text
scripts/
├── build-linux.sh
├── build-windows.ps1
└── build-termux.sh
```

Arquivos de implantação copiados para os releases:

```text
packaging/
├── linux/install-autostart.sh
├── windows/register-autostart.ps1
└── termux/install-autostart.sh
```

---

# Testes

## Testes automatizados

```bash
go test ./...
```

Também é recomendável executar:

```bash
go vet ./...
```

## Testar um downloader sem Telegram

```bash
go run ./cmd/test-download "URL"
```

Esse comando mantém o workspace temporário de propósito para permitir inspeção dos arquivos baixados. O bot normal remove o workspace automaticamente.

## Formatação

Linux/Termux:

```bash
gofmt -w $(find cmd internal -name '*.go')
```

Windows PowerShell:

```powershell
gofmt -w (Get-ChildItem -Recurse -Filter *.go cmd,internal | ForEach-Object FullName)
```

---

# Estratégias por plataforma

## TikTok

1. `yt-dlp` para vídeos.
2. `gallery-dl` como fallback para posts `/photo/`.

O fallback de fotos filtra somente imagens, evitando baixar o áudio associado ao carrossel.

## Instagram

1. `gallery-dl`, opcionalmente autenticado por `INSTAGRAM_COOKIES_FILE`.
2. `yt-dlp` como fallback.

O uso de um arquivo de cookies torna a autenticação portátil entre Linux, Windows e Termux sem depender do perfil local do navegador.

## Threads

Downloader próprio em Go. A página pública é analisada e o post principal é localizado pelo código da URL canônica. O extrator suporta `video_versions`, `image_versions2` e `carousel_media`.

## X / Twitter

1. `gallery-dl`.
2. `yt-dlp` como fallback.

Os arquivos retornados pelo `gallery-dl` são deduplicados por SHA-256 antes do envio.

## Reddit

1. `yt-dlp` para vídeos.
2. RSS público para foto única.

Links de compartilhamento são resolvidos antes da montagem da URL RSS. Galerias continuam marcadas como mídia não suportada.

## YouTube

Somente URLs de Shorts são reconhecidas pelo bot. Vídeos normais do YouTube não fazem parte do escopo atual.

## Erome

Downloader próprio em Go. O HTML do álbum é analisado para localizar imagens e vídeos, com remoção de URLs duplicadas antes do download.

---

# Estrutura do projeto

```text
cmd/
  bot/                 ponto de entrada do bot
  test-download/       teste manual dos downloaders

internal/
  bot/                 polling, fila, workers e envio Telegram
  config/              configuração e .env
  downloader/          manager, fallback e downloaders por plataforma
  media/               modelo de mídia
  platform/            detecção de plataforma
  tools/               detecção de yt-dlp, ffmpeg e gallery-dl
  urlutil/             extração e validação de URLs
  workspace/           diretórios temporários

scripts/                geração dos releases locais
packaging/              scripts de autostart incluídos nos pacotes
```

# Segurança

O `.gitignore` exclui dados sensíveis e artefatos gerados, incluindo:

```text
.env
secrets/
*.cookies.txt
instagram-cookies.txt
bin/
dist/
release/
gallery-dl/
*.log
```

Não coloque tokens, cookies, binários compilados ou downloads de teste no repositório.

# Próximas evoluções

- suporte confiável a galerias do Reddit;
- tratamento automático para mídias acima dos limites da Bot API oficial;
- endpoint configurável para um Telegram Bot API Server próprio;
- automação de releases em CI quando o projeto for publicado/distribuído com mais frequência.
