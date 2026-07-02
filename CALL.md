# Calling (voz / vídeo) — como integrar com whatsgo

whatsgo **não implementa** o protocolo de chamadas. Implementa só o lado
**receive**: parseia `<call>` stanzas e despacha como `events.CallOffer`,
`events.CallAccept`, `events.CallTerminate`, etc., além de expor
`RejectCall` pra recusar. Pra fazer/receber chamadas de verdade (com
áudio e vídeo), use a biblioteca externa
[`purpshell/meowcaller`](https://github.com/purpshell/meowcaller).

---

## Por que meowcaller fica de fora do whatsgo

`meowcaller` importa `go.mau.fi/whatsmeow` direto no código-fonte. Como
whatsgo é um fork que se chama `github.com/thyagodantas/whatsgo`, sem
configuração o Go trata `*whatsmeow.Client` (do upstream) e `*whatsgo.Client`
(do fork) como **tipos diferentes**, mesmo tendo layout idêntico.

Pra integrar os dois sem fork do `meowcaller`, basta uma **`replace` no
`go.mod` do consumidor raiz** (o projeto que importa `whatsgo` e
`meowcaller` juntos):

```go
// go.mod do seu projeto
require (
    github.com/purpshell/meowcaller latest
    github.com/thyagodantas/whatsgo latest
)

// Substitui qualquer referencia a go.mau.fi/whatsmeow pelo nosso fork.
// Como o meowcaller importa go.mau.fi/whatsmeow internamente, isso faz o
// Go resolver pro whatsgo, e os tipos batem em link time.
replace go.mau.fi/whatsmeow => github.com/thyagodantas/whatsgo v0.0.0-XXXXXXXX
```

A versão `v0.0.0-XXXXXXXX` é o SHA atual do whatsgo — pegue em
[github.com/thyagodantas/whatsgo/commits/main](https://github.com/thyagodantas/whatsgo/commits/main).

Depois de `go mod tidy`, você pode usar:

```go
import (
    "github.com/purpshell/meowcaller"
    "github.com/purpshell/meowcaller/audio/malgo"
    "github.com/thyagodantas/whatsgo"
)

wa, _ := whatsgo.NewClient(device, log)
_ = wa.Connect()

// Note: NewCallClient NÃO existe no whatsgo. Você usa meowcaller direto.
// Ambos os clients compartilham o mesmo device store e o mesmo socket.
calls := meowcaller.NewClient(wa.DangerousInternals().RawClient(), meowcaller.WithLogger(*log))

calls.OnIncomingCall(func(call *meowcaller.Call) {
    _ = call.Answer()
    mic, _ := malgo.Mic()
    call.Play(mic)
    spk, _ := malgo.Speaker()
    call.Receive(spk)
})

// Para chamadas de saída:
outbound, err := calls.Call(ctx, "+5511999998888")
_ = outbound
```

> ⚠️ `wa.DangerousInternals().RawClient()` é um helper que **ainda não
> existe** — precisa ser adicionado no whatsgo se você quiser usar esse
> caminho. Alternativa (mais simples, sem patch): construa um
> `*whatsmeow.Client` separado com o mesmo device store e passe pro
> meowcaller, exatamente como o exemplo
> [`examples/cli/main.go`](https://github.com/purpshell/meowcaller/blob/main/examples/cli/main.go)
> do meowcaller faz.

---

## Receber eventos sem meowcaller

Se você só quer **monitorar** chamadas (logar quem ligou, recusar
automaticamente), não precisa de meowcaller. Use direto:

```go
wa.AddEventHandler(func(evt interface{}) {
    switch v := evt.(type) {
    case *events.CallOffer:
        log.Printf("chamada de %s (id=%s, video=%v)", v.From, v.CallID, v.Media)
        _ = wa.RejectCall(ctx, v.From, v.CallID)
    case *events.CallTerminate:
        log.Printf("chamada encerrada: %s", v.Reason)
    }
})
```

`RejectCall` é exposto no `*whatsgo.Client` direto (no `call.go`), sem
precisar do meowcaller.

---

## Roadmap

Se meowcaller se estabilizar como dependência canônica, o whatsgo pode
incorporar:

- **`RawClient()` no `DangerousInternalClient`** — expõe o `*whatsmeow.Client`
  cru pra quem precisa passar pro meowcaller sem abrir um novo socket.
- **Helper `NewCallClient(wa, opts...)`** no pacote raiz — wrapper fino
  que delega pra `meowcaller.NewClient` usando o cast interno. Isso só
  funciona se o consumidor tiver o `replace` no `go.mod`.

Enquanto isso, o caminho mais estável é o descrito acima: dependência
externa no `meowcaller` + `replace` no `go.mod`.