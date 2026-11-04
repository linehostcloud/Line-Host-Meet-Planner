# Line Host® Meet Planner

Plugin para **Mattermost Web e Desktop** que cria reuniões no Google Meet, agenda eventos no Google Calendar e publica lembretes no canal. Cada pessoa usa sua própria Conta Google.

> Integração independente da Line Host®. Não é um produto oficial do Google ou do Mattermost.

## Recursos

- **Iniciar agora:** cria uma sala Google Meet e publica o link na conversa atual.
- **Agendar:** cria um evento no calendário principal do organizador, com link Meet e convites para os endereços informados.
- **Lembretes no canal:** escolha vários avisos entre 1 dia, 2 horas, 1 hora, 30, 15, 10 e 5 minutos antes. O bot menciona o organizador. É possível agendar sem lembretes no Mattermost.
- **Conexão individual:** cada usuário autoriza a própria Conta Google com `/meet connect` e pode remover o token salvo no plugin com `/meet disconnect`.
- **Web e Desktop:** o botão **Meet** aparece junto ao editor de mensagens nos dois clientes.

O monitoramento de salas e a publicação de gravações, transcrições e notas são recursos opcionais, desativados por padrão. Quando ativados, exigem o escopo adicional `meetings.space.readonly` e devem ser avaliados pelo administrador antes de uso.

## Requisitos

- Mattermost **12.0.0 ou superior** em servidor Linux amd64, com `ServiceSettings.SiteURL` apontando para a URL HTTPS pública.
- Go **1.25+** e Node.js **24.13.1** para compilar a partir do código-fonte.
- Projeto Google Cloud com **Google Meet REST API** e **Google Calendar API** ativadas.
- Cliente OAuth 2.0 do tipo **Aplicativo da Web** no projeto Google Cloud do administrador.

O plugin solicita os escopos `https://www.googleapis.com/auth/meetings.space.created` e `https://www.googleapis.com/auth/calendar.events.owned`. A configuração e a eventual verificação do aplicativo OAuth no Google são responsabilidade de quem opera a instância. Não publique o Client Secret, a chave de criptografia ou tokens no GitHub.

## Compilar e instalar

```bash
git clone https://github.com/linehostcloud/Line-Host-Meet-Planner.git
cd Line-Host-Meet-Planner
make dist
```

O pacote gerado fica em `dist/com.linehost.meet-planner-0.1.8.tar.gz`. No Mattermost, acesse **Console do Sistema → Plugins → Gerenciamento de Plugins**, envie o pacote e ative o plugin.

Em **Console do Sistema → Plugins → Line Host Meet Planner**:

1. Copie a **URL de retorno OAuth** exibida pela configuração do plugin.
2. No Google Cloud, adicione essa URL aos **URIs de redirecionamento autorizados** do cliente OAuth. O formato é `https://SEU-MATTERMOST/plugins/com.linehost.meet-planner/api/v1/oauth/callback`.
3. Informe o **Client ID** e o **Client Secret** desse cliente no Mattermost. A chave de criptografia é gerada nas configurações do plugin.
4. Salve as configurações. Cada usuário executa `/meet connect` e autoriza Meet e Calendar na própria Conta Google.

No Desktop, recarregue a janela após a instalação para que o botão **Meet** apareça. Os dados OAuth ficam no servidor Mattermost, não no computador do usuário.

## Uso

Clique em **Meet** ao escrever uma mensagem e escolha **Iniciar agora** ou **Agendar**. Para agendar, preencha título, data, horário e, se necessário, convidados e lembretes. O link e os horários são publicados na conversa selecionada. Os convidados recebem convites pelo Google Calendar.

## Desenvolvimento

```bash
make apply
go test ./server/...
cd webapp
npm ci
npm run check-types
npm run lint
npm run test-ci
```

O código deriva do [Mattermost Google Meet Plugin v2.0.0](https://github.com/mattermost/mattermost-plugin-google-meet), commit `144dec0a0e2c7a53cf8a765471d5cef153f0c1b8`. O Google Calendar Plugin serviu apenas como referência de fluxo. Consulte [LICENSE](LICENSE) e [NOTICE](NOTICE) para a licença e a atribuição das modificações.
