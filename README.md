# Super Picos Downloader

Bot do Telegram em Go que baixa mídias de links e responde no próprio chat. Validado em Windows e Linux `amd64` e Android `arm64` via Termux.

## Plataformas suportadas

| Plataforma | Conteúdo | Estratégia |
| --- | --- | --- |
| TikTok | Vídeos e carrosséis de fotos | `yt-dlp` + `gallery-dl` |
| Instagram | Fotos, vídeos, Reels e carrosséis | `gallery-dl` + cookies, com fallback `yt-dlp` |
| Threads | Fotos, vídeos e carrosséis | Downloader próprio |
| X / Twitter | Fotos, vídeos e álbuns | `gallery-dl` + fallback `yt-dlp` |
| YouTube | Apenas Shorts | `yt-dlp` |
| Reddit | Vídeo ou foto única | `yt-dlp` + RSS |
| Erome | Fotos e vídeos | Downloader próprio |

Galerias do Reddit ainda têm limitações. Conteúdos privados, removidos ou que exigem autenticação podem falhar. No Instagram, os cookies devem usar o formato Netscape/Mozilla, sem depender do perfil local do navegador.

## Funcionamento e limites

- Responde à mensagem original e mostra um status temporário durante o download.
- Descarta mensagens pendentes ao iniciar e processa os novos links por fila, com múltiplos workers.
- Preserva ordem e legendas dos álbuns, dividindo-os em grupos de até 10 mídias, sem grupos unitários.
- Vídeos de até 50.000.000 bytes seguem sem recompressão. Acima disso, tenta reduzir para **até 48 MB**, com no máximo três tentativas usando ffmpeg.
- A compressão usa `veryfast`, limita a maior dimensão e ajusta novas tentativas pelo tamanho obtido. Pode reduzir a qualidade, mas não corta a duração.
- Prepara todos os vídeos antes de enviar o álbum. Se não atingir a margem segura, informa o problema e não envia o resultado.
- Cada download tem prazo de 3 minutos; a preparação dos vídeos tem prazo separado de 10 minutos para o conjunto.
- Remove os arquivos temporários ao terminar, inclusive em caso de erro. Os logs incluem os tempos de compressão e de preparação/envio.

Fotos não são recomprimidas e continuam sujeitas aos limites da [Bot API oficial](https://core.telegram.org/bots/api). Um [servidor local da Bot API](https://github.com/tdlib/telegram-bot-api) não faz parte da configuração padrão.

## Requisitos e configuração

- Go **1.22.2+** para executar pelo código-fonte ou compilar.
- `yt-dlp`, `gallery-dl` e `ffmpeg` disponíveis no `PATH` do processo do bot.
- FFmpeg com os encoders `libx264` e `aac` para recompressão.

Na pasta do projeto ou do pacote extraído, crie o `.env` apenas na primeira configuração:

```bash
# Linux / Termux
cp .env.example .env
```

```powershell
# Windows / PowerShell
Copy-Item .env.example .env
```

Edite o arquivo:

```env
BOT_TOKEN=SEU_TOKEN_DO_BOTFATHER
MAX_WORKERS=3
INSTAGRAM_COOKIES_FILE=secrets/instagram-cookies.txt
```

`BOT_TOKEN` é obrigatório; `MAX_WORKERS` controla os downloads simultâneos (padrão: 3). Os cookies são opcionais, mas necessários para muitos conteúdos do Instagram. `.env` e `secrets/` são ignorados pelo Git: nunca publique tokens ou cookies.

Depois de instalar as dependências da sua plataforma, valide:

```bash
go version
yt-dlp --version
gallery-dl --version
ffmpeg -version
```

## Linux

### Instalar e executar

Em Ubuntu, Debian, KDE neon e derivados, instale Go 1.22.2+ pelo método da distribuição e execute:

```bash
sudo apt update
sudo apt install -y ffmpeg python3 python3-pip pipx
pipx ensurepath
pipx install "yt-dlp[default,curl-cffi]"
pipx install gallery-dl
```

Reabra o terminal para atualizar o `PATH`. Na raiz do projeto, após configurar o `.env`:

```bash
go mod download
go run ./cmd/bot
```

Para compilar e executar diretamente:

```bash
mkdir -p bin
go build -trimpath -ldflags="-s -w" -o bin/super-picos ./cmd/bot
./bin/super-picos
```

### Pacote e inicialização automática

Na raiz do projeto:

```bash
./scripts/build-linux.sh
```

Saída: `dist/super-picos-linux-amd64.tar.gz`. Extraia em uma pasta permanente, configure o `.env` e os cookies e, dentro dessa pasta, execute:

```bash
./install-autostart.sh
```

O instalador cria e inicia o serviço `systemd --user` chamado `super-picos.service`:

```bash
# Status e logs
systemctl --user status super-picos.service
journalctl --user -u super-picos.service -f

# Iniciar ou parar manualmente
systemctl --user start super-picos.service
systemctl --user stop super-picos.service

# Permitir início no boot, antes do login
sudo loginctl enable-linger "$USER"
```

## Windows

### Instalar e executar

No PowerShell:

```powershell
winget install -e --id GoLang.Go
winget install -e --id Gyan.FFmpeg
winget install -e --id Python.Python.3.13
```

Reabra o terminal e instale os downloaders:

```powershell
py -m pip install --user --upgrade "yt-dlp[default,curl-cffi]" gallery-dl
```

Se os comandos não forem encontrados, adicione a pasta de scripts do Python ao `PATH` do usuário:

```powershell
$Scripts = py -c "import sysconfig; print(sysconfig.get_path('scripts', scheme='nt_user'))"
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($userPath -split ';') -notcontains $Scripts) {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$Scripts", "User")
}
```

Reabra o PowerShell após alterar o `PATH`. Na raiz do projeto, com o `.env` configurado:

```powershell
cd C:\Users\dells\Projetos\super-picos-downloader
go mod download
go run .\cmd\bot
```

Ajuste o caminho se o projeto estiver em outra pasta. Mantenha o terminal aberto e use **Ctrl+C** para parar.

### Pacote e inicialização automática

Na raiz do projeto:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1
```

Saída: `dist\super-picos-windows-amd64.zip`, com `super-picos.exe` (console e logs) e `super-picos-hidden.exe` (sem janela, para autostart).

Extraia em uma pasta permanente, configure o `.env` e os cookies e, dentro dessa pasta, execute:

```powershell
powershell -ExecutionPolicy Bypass -File .\register-autostart.ps1
```

O script registra e inicia a tarefa **Super Picos Downloader**, configurada para iniciar no login:

```powershell
# Iniciar manualmente
Start-ScheduledTask -TaskName "Super Picos Downloader"

# Diagnóstico
Get-ScheduledTaskInfo -TaskName "Super Picos Downloader" |
    Format-List LastRunTime,LastTaskResult
Get-Process super-picos-hidden

# Parar a tarefa
Stop-ScheduledTask -TaskName "Super Picos Downloader"

# Se o processo continuar ativo
Stop-Process -Name super-picos-hidden -Force

# Remover a inicialização automática
Unregister-ScheduledTask -TaskName "Super Picos Downloader" -Confirm:$false
```

Para diagnóstico com logs no terminal, pare a tarefa e execute `.\super-picos.exe` na pasta instalada.

## Android / Termux

### Instalar e executar

Mantenha o projeto e o executável dentro do `$HOME` do Termux, evitando `/sdcard` e `~/storage/shared`.

```bash
pkg update && pkg upgrade -y
pkg install -y golang python ffmpeg git python-yt-dlp
python -m pip install --upgrade gallery-dl
```

Valide as ferramentas com os comandos da seção de requisitos e `python --version`. Na raiz do projeto, após configurar o `.env`:

```bash
go run ./cmd/bot
```

Para compilar nativamente e executar:

```bash
mkdir -p bin
go build -trimpath -ldflags="-s -w" -o bin/super-picos ./cmd/bot
./bin/super-picos
```

### Pacote e inicialização com Termux:Boot

Na raiz do projeto:

```bash
./scripts/build-termux.sh
```

A arquitetura é detectada pelo Go. Em ARM64, a saída é `dist/super-picos-termux-arm64.tar.gz`.

Instale o **Termux:Boot da mesma origem/assinatura do Termux** e abra-o pelo menos uma vez. Extraia o pacote em uma pasta permanente dentro do `$HOME`, configure o `.env` e os cookies e execute nessa pasta:

```bash
./install-autostart.sh
```

O instalador cria `~/.termux/boot/start-super-picos` para executar no boot. O script usa `termux-wake-lock` quando disponível e grava `bot.log` na pasta instalada:

```bash
tail -f bot.log
```

Deixe **Termux e Termux:Boot sem restrição de bateria** para reduzir o risco de encerramento pelo Android.

## Atualizações e pacotes

Os scripts geram os pacotes em `dist/`, incluindo `.env.example`, este README, o instalador de autostart e `secrets/README.txt`. Eles não copiam seu `.env` nem cookies reais.

Use uma pasta de instalação fora de `dist/`: os scripts recriam a pasta de saída ao gerar um pacote. Para atualizar, pare o bot, substitua os executáveis e preserve o `.env` e os cookies. Se usa `go run`, basta parar e iniciar novamente após alterar o código.

Execute apenas uma instância por token. No terminal, use **Ctrl+C** para parar; no Linux/Termux, o bot também trata `SIGTERM`.

## Desenvolvimento e testes

```bash
go test ./...
go vet ./...
```

Mantenha os arquivos `*_test.go` no projeto: eles não entram no executável. Os testes unitários de compressão simulam o ffmpeg.

Para testar um download sem enviar ao Telegram:

```bash
go run ./cmd/test-download "URL"
```

Esse comando preserva os arquivos baixados para inspeção; remova-os quando terminar.

Formatação no Linux/Termux:

```bash
gofmt -w $(find cmd internal -name '*.go')
```

No PowerShell:

```powershell
gofmt -w (Get-ChildItem -Recurse -Filter *.go cmd,internal | ForEach-Object FullName)
```

## Estrutura e próximos passos

| Pasta | Responsabilidade |
| --- | --- |
| `cmd/` | Bot e teste manual de download |
| `internal/bot/` | Fila, workers, compressão e envio |
| `internal/downloader/` | Downloaders e estratégias alternativas |
| Demais pastas de `internal/` | Configuração, mídias, URLs, ferramentas e temporários |
| `scripts/` | Geração dos pacotes |
| `packaging/` | Instaladores de inicialização automática |

Próximas melhorias: galerias do Reddit, endpoint configurável da Bot API e automação de releases. Não versione tokens, cookies, binários, logs ou downloads de teste.
