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
- Antes de iniciar os workers, limpa a fila pendente do Telegram. Comandos e links com data anterior ao início também são ignorados se forem entregues depois. A comparação usa a precisão de segundos do Telegram e depende do relógio correto do sistema.
- Preserva ordem e legendas dos álbuns, dividindo-os em grupos de até 10 mídias, sem grupos unitários.
- Por padrão, vídeos de até 50.000.000 bytes seguem sem recompressão. Acima disso, tenta reduzir para **até 48 MB**, com no máximo três tentativas usando ffmpeg.
- A compressão usa `veryfast`, limita a maior dimensão e ajusta novas tentativas pelo tamanho obtido. Pode reduzir a qualidade, mas não corta a duração.
- Prepara todos os vídeos antes de enviar o álbum. Se não atingir a margem segura, informa o problema e não envia o resultado.
- Cada download tem prazo de 3 minutos; a preparação dos vídeos tem prazo separado de 10 minutos para o conjunto.
- Remove os arquivos temporários ao terminar, inclusive em caso de erro. Os logs incluem os tempos de compressão e de preparação/envio.
- Os logs do bot aparecem em blocos com data, horário e nível, detalhes abaixo e erros separados por linha. Eventos de workers diferentes não se misturam dentro do mesmo bloco. Mensagens comuns do chat não são registradas.

Por padrão, fotos não são recomprimidas e continuam sujeitas aos limites da [Bot API oficial](https://core.telegram.org/bots/api). Um [servidor local da Bot API](https://github.com/tdlib/telegram-bot-api) não faz parte da configuração padrão.

- Reaproveita conexões HTTP entre envios. O log `envio ao Telegram concluído` registra tipo, quantidade, bytes e tempo da requisição, incluindo o processamento do Telegram, separado da preparação. Essa otimização não reduz a qualidade nem garante aumento da velocidade da rede.

## Requisitos e configuração

Para VPS Linux com 1 GB de RAM, defina `LOW_MEMORY=true` no `.env`. Esse modo força um worker (mesmo com `MAX_WORKERS` maior), limita decodificação, codificação e filtros do ffmpeg a uma thread e aplica um limite suave de 256 MiB ao runtime Go, salvo se `GOMEMLIMIT` já estiver definido. Mantém a qualidade e os limites de tamanho da compressão; a conversão individual pode demorar mais. Os processos externos e o sistema operacional não estão incluídos no limite do Go, portanto não é garantia contra falta de memória.

No Linux, quando `TMPDIR` não está definido, o modo usa `data/tmp` no disco para os downloads e processos filhos, evitando o `/tmp` em RAM. Se definir `TMPDIR`, use um diretório existente em disco. Execute o binário compilado na VPS, em vez de `go run`; para compilar com menos concorrência: `GOTMPDIR="$HOME/.cache/go-tmp" CGO_ENABLED=0 go build -p 1 -trimpath -o super-picos ./cmd/bot`, criando antes essa pasta. Após atualizar o binário e o `.env`, reinicie o serviço.

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
REDUCE_MEDIA=true
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

## Redução moderada para envio

A configuração acima mantém `REDUCE_MEDIA=true` no `.env`, ativando a redução nas próximas inicializações, inclusive pela tarefa automática. O modo tenta diminuir fotos e vídeos antes do upload:

- Fotos a partir de 256 KB: até 1920 pixels no lado maior, JPEG qualidade 85; transparência vira fundo branco.
- Vídeos de 5 MB até 50 MB: até 1280 pixels no lado maior, H.264 CRF 26 e áudio AAC 128 kb/s, sem cortar a duração.
- Usa a cópia somente se economizar pelo menos 10% do tamanho. Se a redução opcional falhar, mantém o original. Cancelamentos interrompem a preparação.
- Arquivos menores são preservados; vídeos acima de 50 MB seguem diretamente para a compressão obrigatória já descrita.

A resolução não é ampliada. Álbuns mantêm ordem e legendas. A redução pode perder detalhes e levar tempo de processamento; compare o tempo total, não apenas o upload.

Para experimentar no PowerShell, pare a instância atual e execute na raiz do projeto:

```powershell
$env:REDUCE_MEDIA="true"
go run .\cmd\bot
```

Para desativar permanentemente, altere para `REDUCE_MEDIA=false` no `.env` e reinicie o bot. Sem essa variável, o padrão do código é desativado. Alterar apenas o `.env` não exige recompilação, desde que o executável já inclua esse recurso.

Variáveis definidas no terminal têm prioridade sobre o `.env`. Para voltar a usar o valor do arquivo no PowerShell, execute `Remove-Item Env:REDUCE_MEDIA -ErrorAction SilentlyContinue` antes de iniciar o bot. No Linux/Termux, um teste temporário pode ser feito com `REDUCE_MEDIA=true go run ./cmd/bot`.

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

## Minigame do grupo

O minigame usa SQLite sem CGO (`modernc.org/sqlite`) em `data/minigame.db`, relativo ao diretório de execução. A pasta é criada automaticamente. Preserve essa pasta entre atualizações; para backup simples, pare o bot e copie a pasta inteira. O banco não integra os pacotes de release.

- `/grow`: uma tentativa a cada 6 horas, contadas desde a última tentativa no grupo, inclusive quando o resultado é zero. Reiniciar o bot ou virar o dia não libera antecipadamente. A data local em `America/Fortaleza` continua sendo usada para as sequências de dias (base de fusos embutida).
- `/rank`: ranking do grupo com nomes visíveis; `[+]` já pode tentar crescer novamente e `[~]` último crescimento há mais de sete datas locais. Quem nunca cresceu recebe `[+]`. Empates seguem o ID numérico do usuário. Rankings longos são enviados em várias mensagens.
- `/duelo valor`: aposta inteira de pelo menos 1 cm; outro membro aceita com **Ataque!**. Não reserva saldo: ambos os saldos são revalidados ao aceitar. Convites persistem após reiniciar, não expiram e são invalidados por `/limpar confirmar`. Só a primeira aceitação válida conclui a aposta. O vencedor é sorteado com `crypto/rand`. Em teste, a sequência atual de vitórias define o peso: 0–2 → 100; 3–4 → 85; 5–6 → 70; 7 ou mais → 55. A chance é o peso do jogador dividido pela soma dos dois pesos. Contra alguém com peso 100, isso representa 50%, 45,9%, 41,2% ou 35,5%. Pesos iguais resultam em 50/50. Uma derrota zera a sequência e remove a penalidade. Tamanho, posição, autoria do convite e winrate não influenciam. Sequências de vitórias ainda são possíveis; não há derrota garantida. As chances anteriores ao sorteio aparecem na resposta quando há penalidade. A regra vale na aceitação, inclusive para convites antigos; as sequências já existentes são consideradas. Se o sorteio falhar, a aposta permanece aberta e os saldos não mudam.
- `/emprestimo`: disponível apenas após pelo menos uma tentativa de crescimento no grupo, com tamanho atual de 0 cm e sem dívida. A primeira tentativa pode resultar em zero. Após limpar o grupo, é preciso tentar crescer novamente. Sorteia 0–16 cm; zero não gera dívida. No `/grow`, o pagamento é o menor entre dívida, crescimento bruto e sorteio de 1–3 cm. A mensagem mostra crescimento líquido e valor pago.
- `/doar quantidade @usuário` (temporário): transfere centímetros entre membros do grupo, sem alterar dívida, cooldown ou estatísticas de duelos. Ex.: `/doar 5 @pedro`. Exige inteiro positivo e saldo suficiente; não permite doar para si mesmo ou bots. O destinatário deve ter enviado uma mensagem após a atualização (por exemplo, `/status`), para o bot conhecer seu @. A identidade é armazenada por grupo e a participação e o @ atual são revalidados antes da transferência. Essa verificação é garantida quando o bot é administrador. A transferência é transacional e não cria centímetros. A remoção futura do comando não desfaz doações já feitas.
- `/status`: tamanho, rank, sequências, duelos, vitórias, win rate, cm ganhos/perdidos em duelos e dívida. Win rate é arredondado ao inteiro mais próximo; sequências de crescimento incluem dias com resultado zero.
- `/limpar confirmar`: apenas administradores identificados por conta pessoal. Zera estatísticas e remove histórico de crescimento e convites apenas do grupo atual, preservando nomes dos participantes. Também remove o intervalo pendente, permitindo crescer imediatamente após a limpeza.

Pesos por resultado de crescimento/empréstimo: 0–2 têm peso 2 cada; 3–8 peso 10 cada; 9–13 peso 3 cada; 14–16 peso 1 cada (84 no total). Nomes são atualizados quando a pessoa participa; o bot não tenta enumerar todos os membros do Telegram. Os dados usam chat ID + user ID. Transações SQLite protegem alterações. Novos registros de crescimento usam timestamps completos na chave do histórico, permitindo várias tentativas no mesmo dia. Sequências continuam contando dias, sem aumentar por tentativas no mesmo dia.

A atualização preserva saldos, dívidas, histórico, sequências e convites. Registros antigos têm somente a data do último crescimento: cada jogador pode tentar uma vez após a atualização para começar a registrar o horário preciso, e depois deve aguardar 6 horas. Para atualizar a VPS, pare o serviço, faça `git pull --ff-only`, recompile o binário e inicie o serviço. Apenas `git pull` não atualiza o executável em execução. Preserve `data/` e o `.env`.

Para validar: `go test ./...` e `go vet ./...`. Os scripts de build existentes continuam válidos. A validação nativa do Termux deve ser feita no dispositivo. A consulta de participação de outros usuários pelo Telegram é garantida quando o bot é administrador do grupo; conceda esse papel para garantir a verificação do botão de duelo.

### Publicações sem mídia

Links de plataformas suportadas que retornam ausência de mídia são ignorados, sem resposta ou mensagem temporária de download. O status de preparação aparece somente depois de obter mídia. Em uma cadeia de tentativas, resultados sem mídia e estratégias incompatíveis podem ser ignorados juntos; erros de acesso, autenticação, formato não suportado, indisponibilidade ou download continuam sendo informados, mesmo quando outra tentativa não encontra mídia.
