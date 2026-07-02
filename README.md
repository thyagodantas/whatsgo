# whatsgo

> Engine de protocolo do **WhatsApp Web Multi-device** em Go — escrita do
> zero, com suporte nativo a mensagens interativas (botões, listas, native_flow).

[![Go Reference](https://pkg.go.dev/badge/github.com/thyagodantas/whatsgo.svg)](https://pkg.go.dev/github.com/thyagodantas/whatsgo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

`whatsgo` é uma biblioteca Go que fala nativamente o protocolo do WhatsApp
Web Multi-device. Ela cuida de toda a complexidade do transporte (Noise
handshake, criptografia Signal, sessões multi-device, filas de envio,
retry receipts) e expõe uma API limpa pra você parear via QR e
enviar/receber mensagens.

---

## O que o `whatsgo` resolve

Três problemas recorrentes ao enviar mensagens interativas contra o servidor
atual do WhatsApp são tratados **dentro da engine**, sem workaround no
consumidor:

| # | Problema | Solução nativa |
| --- | --- | --- |
| 1 | Versão antiga no handshake → servidor recusa interativos com **405** | `WAVersion` recente fixada no `store` |
| 2 | Stanza `<biz>` placeholder → servidor recusa com 405 | Stanza `<biz>` no formato `native_flow` montada automaticamente |
| 3 | Sem `MessageSecret` (32 bytes) → destinatário não descriptografa | Secret injetado automaticamente em qualquer mensagem interativa |

Quem consome a biblioteca não monta `<biz>` na mão, não gera `MessageSecret`,
não mexe em versão. Chama `SendButtons`, `SendList` ou `SendInteractive` e
funciona.

---

## Mensagens interativas (destaque)

São o carro-chefe do projeto. Todas funcionam em **conta pessoal** (não
precisam de aprovação Business API).

### Botões de resposta rápida

```go
import (
    "github.com/thyagodantas/whatsgo"
)

client.SendButtons(ctx, jid,
    "Escolha uma opção:",     // texto
    "rodapé opcional",        // footer
    []whatsgo.ButtonSpec{
        {ID: "sim", Text: "Sim"},
        {ID: "nao", Text: "Não"},
        {ID: "talvez", Text: "Talvez"},
    },
)
```

Limite do WhatsApp: até **3 botões** por mensagem. Cada toque do destinatário
chega como evento `Message` com `InteractiveResponse` ou
`ButtonsResponseMessage`, já pronto pra você filtrar por `ID`.

### Menu em lista (single_select)

```go
client.SendList(ctx, jid,
    "Cardápio",              // título do header
    "Selecione um item",     // descrição (body)
    "Ver opções",            // texto do botão que abre a lista
    "rodapé opcional",
    []whatsgo.SectionSpec{
        {
            Title: "Bebidas",
            Rows: []whatsgo.RowSpec{
                {ID: "cafe", Title: "Café", Description: "Expresso"},
                {ID: "cha", Title: "Chá", Description: "Mate"},
            },
        },
        {
            Title: "Comidas",
            Rows: []whatsgo.RowSpec{
                {ID: "pao", Title: "Pão de queijo", Description: "10 un."},
            },
        },
    },
)
```

Suporta **N seções**, cada uma com **N linhas**. Limite prático recomendado
de até 10 linhas totais por lista (UX do WhatsApp).

### Native flow genérico (CTAs, quick_reply, payments)

```go
client.SendInteractive(ctx, jid,
    "Confira nossa oferta",            // body
    "rodapé opcional",
    []whatsgo.NativeFlowSpec{
        {
            Name: "cta_url",
            ParamsJSON: `{"display_text":"Abrir site","url":"https://exemplo.com"}`,
        },
        {
            Name: "cta_call",
            ParamsJSON: `{"display_text":"Ligar agora","phone_number":"+5511999998888"}`,
        },
        {
            Name: "cta_copy",
            ParamsJSON: `{"display_text":"Copiar cupom","id":"CUPOM10"}`,
        },
        {
            Name: "quick_reply",
            ParamsJSON: `{"display_text":"Falar com atendente","id":"qr_atendente"}`,
        },
    },
)
```

`Name` aceita qualquer fluxo suportado pelo WA Web (`cta_url`, `cta_call`,
`cta_copy`, `quick_reply`, `review_and_pay`, …). `ParamsJSON` é o JSON de
parâmetros no formato exato esperado pelo servidor pra cada tipo.

---

## API completa — principais funções

### Conexão e pareamento

| Função | O que faz |
| --- | --- |
| `NewClient(deviceStore, logger)` | Cria um client a partir de um device store |
| `Connect()` / `ConnectContext(ctx)` | Abre o socket Noise e inicia a sessão |
| `Disconnect()` | Fecha o socket (mantém a sessão salva) |
| `IsConnected()` | Socket aberto? |
| `IsLoggedIn()` | Sessão pareada ativa? |
| `Logout(ctx)` | Desfaz o pareamento (apaga credenciais) |
| `GetQRChannel(ctx)` | Canal pra consumir eventos de QR durante pareamento |
| `WaitForConnection(timeout)` | Bloqueia até conectar ou estourar timeout |

### Envio de mensagens

| Função | O que faz |
| --- | --- |
| `SendMessage(ctx, to, msg, extra...)` | Envio genérico de qualquer `waE2E.Message` |
| `SendButtons(ctx, to, text, footer, buttons)` | Botões de resposta rápida (até 3) |
| `SendList(ctx, to, title, desc, btn, footer, sections)` | Menu em lista |
| `SendInteractive(ctx, to, body, footer, buttons)` | Native flow genérico (CTAs, payments…) |
| `RevokeMessage(ctx, chat, id)` | Apaga uma mensagem (pra todos, quando permitido) |
| `BuildEdit(chat, id, newContent)` | Edita uma mensagem enviada |
| `BuildReaction(chat, sender, id, reaction)` | Reage a uma mensagem |
| `SetDisappearingTimer(ctx, chat, timer, ts)` | Define timer de mensagens temporárias |

### Mídia (upload / download)

| Função | O que faz |
| --- | --- |
| `Upload(ctx, data, MediaType)` | Faz upload do arquivo e devolve URL + chaves |
| `UploadReader(ctx, reader, tempFile, MediaType)` | Upload streamed |
| `UploadNewsletter(ctx, data, MediaType)` | Upload específico pra newsletter |
| `Download(ctx, msg)` | Baixa mídia de uma mensagem |
| `DownloadThumbnail(ctx, msg)` | Baixa só a miniatura |
| `DownloadMediaWithPath(ctx, directPath)` | Baixa por path direto |
| `DeleteMedia(ctx, type, directPath, hash, handle)` | Remove mídia do servidor |

### Eventos (recebimento)

| Função | O que faz |
| --- | --- |
| `AddEventHandler(handler)` | Registra callback que recebe todo evento |
| `RemoveEventHandler(id)` | Remove handler por ID |
| Eventos disponíveis | `Message`, `Receipt`, `Presence`, `Connected`, `PairSuccess`, `Disconnect`, `GroupInfo`, `JoinedGroup`, `Picture`, `PrivacySettings`, `AppState`, `Blocklist`, … |

### Grupos

| Função | O que faz |
| --- | --- |
| `CreateGroup(ctx, ReqCreateGroup)` | Cria grupo novo |
| `JoinGroupWithLink(ctx, code)` | Entra via link de convite |
| `LeaveGroup(ctx, jid)` | Sai do grupo |
| `GetGroupInfo(ctx, jid)` | Metadados + participantes |
| `GetGroupInviteLink(ctx, jid, reset)` | Gera/renova link |
| `UpdateGroupParticipants(ctx, jid, list, action)` | Add/remove/promote |
| `SetGroupName/Topic/Photo/Description/Announce/Locked/JoinApproval/MemberAddMode` | Configurações |
| `GetJoinedGroups(ctx)` | Lista grupos do user |
| `GetLinkedGroupsParticipants(ctx, community)` | Lista subgrupos de comunidade |

### Presença / digitação

| Função | O que faz |
| --- | --- |
| `SendPresence(ctx, state)` | online/offline |
| `SendChatPresence(ctx, jid, state, media)` | "digitando…" / "gravando áudio…" |
| `SubscribePresence(ctx, jid)` | Pede atualização de presença de alguém |

### Receipts (entregue / lido)

| Função | O que faz |
| --- | --- |
| `MarkRead(ctx, ids, ts, chat, sender, types...)` | Marca mensagens como lidas |
| `SetForceActiveDeliveryReceipts(bool)` | Força delivery receipts ativos |

### Contatos, perfil e usuários

| Função | O que faz |
| --- | --- |
| `IsOnWhatsApp(ctx, phones)` | Verifica se números têm WhatsApp |
| `GetUserInfo(ctx, jids)` | Puxa info de perfil (push name, foto…) |
| `GetProfilePictureInfo(ctx, jid, params)` | Foto de perfil |
| `GetBlocklist(ctx)` / `UpdateBlocklist(ctx, jid, action)` | Bloqueios |
| `GetContactQRLink(ctx, revoke)` | QR code de contato |
| `ResolveContactQRLink(ctx, code)` | Resolve QR code de contato |
| `ResolveBusinessMessageLink(ctx, code)` | Resolve link de mensagem comercial |
| `GetBusinessProfile(ctx, jid)` | Perfil business |
| `GetUserDevices(ctx, jids)` | Lista devices de um user (multi-device) |

### Privacidade

| Função | O que faz |
| --- | --- |
| `GetPrivacySettings(ctx)` | Puxa configs atuais |
| `SetPrivacySetting(ctx, name, value)` | Altera (last seen, foto, status…) |
| `SetDefaultDisappearingTimer(ctx, timer)` | Timer padrão de mensagens temporárias |

### Status / newsletter

| Função | O que faz |
| --- | --- |
| `SetStatusMessage(ctx, msg)` | Publica status (texto) |
| `GetStatusPrivacy(ctx)` | Privacidade do status |
| `SendAppState(ctx, patch)` | Envia patch de app state |

### Newsletter (canais)

| Função | O que faz |
| --- | --- |
| `UploadNewsletter(ctx, data, MediaType)` | Upload pra newsletter |
| (envio em `send.go`) | `sendNewsletter(ctx, …)` para mídias em canal |

### Retry & resiliência

| Função | O que faz |
| --- | --- |
| `SetMaxParallelRetryReceiptHandling(n)` | Limita concorrência de retries |
| Retry receipts automáticos | Quando o destinatário falha ao descriptografar, o whatsgo reenvia sozinho usando o cache interno (`RecentMessage`) |

---

## Exemplo completo

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/thyagodantas/whatsgo"
    "github.com/thyagodantas/whatsgo/store/sqlstore"
    "github.com/thyagodantas/whatsgo/types"
    "github.com/thyagodantas/whatsgo/types/events"
    "github.com/thyagodantas/whatsgo/proto/waE2E"
    "google.golang.org/protobuf/proto"
    waLog "github.com/thyagodantas/whatsgo/util/log"
)

func main() {
    dbLog := waLog.Stdout("DB", "INFO", true)
    container, _ := sqlstore.New("sqlite3", "file:store.db?_pragma=foreign_keys(1)", dbLog)
    deviceStore := container.NewDevice()

    client := whatsgo.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))

    client.AddEventHandler(func(evt interface{}) {
        switch v := evt.(type) {
        case *events.Message:
            if v.Message.GetConversation() != "" {
                fmt.Printf("[%s] %s\n", v.Info.Sender, v.Message.GetConversation())
            }
            if resp := v.Message.GetInteractiveResponseMessage(); resp != nil {
                fmt.Printf("[%s] clicou em: %s\n", v.Info.Sender, resp.GetNativeFlowResponseMessage().GetSelectedButtonID())
            }
        case *events.Receipt:
            fmt.Printf("receipt: %v em %s\n", v.Type, v.Chat)
        }
    })

    if client.Store.ID == nil {
        qrChan, _ := client.GetQRChannel(context.Background())
        if err := client.Connect(); err != nil {
            panic(err)
        }
        for evt := range qrChan {
            if evt.Event == "code" {
                fmt.Println("QR code:", evt.Code)
                _ = os.WriteFile("qr.png", mustQRPNG(evt.Code), 0644)
            } else if evt.Event == "success" {
                break
            }
        }
    } else {
        _ = client.Connect()
    }

    jid, _ := types.ParseJID("5511999998888@s.whatsapp.net")

    // texto
    client.SendMessage(context.Background(), jid, &waE2E.Message{
        Conversation: proto.String("Olá!"),
    })

    // botões
    client.SendButtons(context.Background(), jid,
        "Escolha:", "",
        []whatsgo.ButtonSpec{
            {ID: "sim", Text: "Sim"},
            {ID: "nao", Text: "Não"},
        })

    // lista
    client.SendList(context.Background(), jid,
        "Cardápio", "Selecione", "Ver opções", "",
        []whatsgo.SectionSpec{{
            Title: "Bebidas",
            Rows: []whatsgo.RowSpec{
                {ID: "cafe", Title: "Café", Description: "Expresso"},
            },
        }})
}
```

---

## Pacotes

| Pacote | Conteúdo |
| --- | --- |
| `whatsgo` | Cliente principal (`Client`, `SendMessage`, `SendButtons`, …) |
| `store` | Interfaces de persistência de chaves, identidade, sessões |
| `store/sqlstore` | Implementação SQL (SQLite/MySQL/Postgres) |
| `socket` | Transporte Noise + frames |
| `binary` | Encoder/decoder de nodes WA (stanzas XML-like) |
| `proto` | Definições protobuf (`waE2E`, `waCommon`, `waAppStateSync`, …) |
| `types` | Tipos públicos (`JID`, `MessageID`, `UserInfo`, `GroupInfo`, …) |
| `types/events` | Eventos emitidos pelo client |
| `appstate` | Sincronização de app state (contatos, pinos, mutes…) |
| `util/log` | Logger |
| `util/{cbcutil,gcmutil,hkdfutil,keys}` | Helpers de criptografia |

---

## Build & dev

```bash
go build ./...     # compila tudo
go vet ./...       # checa estática
go test ./...      # roda os testes
```

---

## Sobre

- **Módulo Go:** `github.com/thyagodantas/whatsgo`
- **Licença:** MIT — ver [`LICENSE`](LICENSE)
- Mantido por [@thyagodantas](https://github.com/thyagodantas)