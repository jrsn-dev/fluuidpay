# Retries e Dead Letter Queue em Go — Mensageria de Pagamentos

**Versão:** 1.0.0  
**Escopo:** evento assíncrono `PaymentProcessed` entre Payment Service e Carrinho  
**Implementação de referência:** RabbitMQ com `github.com/rabbitmq/amqp091-go`  
**Fonte de contexto:** conversa compartilhada [1]

## 1. Objetivo

Este documento especifica uma implementação operacional de retries e Dead Letter Queue (DLQ) para eventos de pagamento. O desenho assume que o Payment Service persiste o pagamento e cria um evento Outbox; um publicador envia o evento ao broker; o consumidor do Carrinho processa a mensagem de forma idempotente e confirma o recebimento somente depois de concluir o efeito de negócio.

O mecanismo diferencia falhas **transitórias**, que podem ser corrigidas por nova tentativa, de falhas **permanentes**, que exigem correção de dados, contrato ou intervenção operacional. A mensagem nunca deve ser descartada silenciosamente.

## 2. Topologia RabbitMQ

A topologia utiliza uma exchange principal, filas de retry com TTL e uma DLQ final. Cada fila de retry possui `x-message-ttl` e `x-dead-letter-exchange` apontando de volta para a exchange principal. Assim, uma mensagem aguardará o intervalo configurado e será roteada novamente para a fila de consumo.

```text
payment.events (topic exchange)
        |
        +-- routing key payment.processed --> payment.processed
        |                                      |
        |                                      +-- consumidor Carrinho
        |                                      |
        |                                      +-- rejeição transitória
        |                                             |
        |                                             v
        |                              payment.processed.retry.1 (10s)
        |                              payment.processed.retry.2 (60s)
        |                              payment.processed.retry.3 (300s)
        |                                             |
        |                                             v
        +-------------------------------------- payment.processed

        rejeição permanente ou limite excedido
                         |
                         v
              payment.processed.dlq
```

| Recurso | Tipo | Função |
|---|---|---|
| `payment.events` | `topic` exchange | Roteia eventos de pagamento |
| `payment.processed` | fila durável | Consumo normal pelo Carrinho |
| `payment.processed.retry.1` | fila durável | Aguarda 10 segundos |
| `payment.processed.retry.2` | fila durável | Aguarda 60 segundos |
| `payment.processed.retry.3` | fila durável | Aguarda 5 minutos |
| `payment.processed.dlq` | fila durável | Armazena mensagens não processáveis |
| `payment.dlx` | `direct` exchange | Roteia mensagens para retry ou DLQ |

## 3. Classificação de falhas

A classificação deve ser determinística e testável. O consumidor deve evitar retry de mensagens inválidas, porque repetir um payload malformado apenas aumenta a latência e o custo operacional.

| Classe | Exemplos | Ação |
|---|---|---|
| Transitória | timeout de banco, conexão perdida, indisponibilidade temporária do Carrinho | Retry com backoff |
| Permanente | JSON inválido, schema incompatível, campo obrigatório ausente | Publicar na DLQ |
| Conflito idempotente | evento já processado | Confirmar sem reaplicar efeito |
| Erro de negócio | pedido cancelado ou estado incompatível | Confirmar após registrar rejeição, ou DLQ conforme contrato |
| Erro desconhecido | panic recuperado ou erro não classificado | Retry limitado; depois DLQ |

O consumidor não deve utilizar `Nack(requeue=true)` indefinidamente. Isso cria um ciclo quente, pressiona o broker e impede que outras mensagens avancem. O retry deve ser controlado pela contagem de tentativas no cabeçalho `x-retry-count` ou por headers equivalentes.

## 4. Contrato do evento

```json
{
  "event_id": "evt-123",
  "event_type": "PaymentProcessed",
  "schema_version": 1,
  "occurred_at": "2026-08-10T14:00:00Z",
  "transaction_id": "txn-789",
  "order_id": "order-456",
  "user_id": "user-123",
  "status": "APPROVED",
  "amount_minor": 19990,
  "currency": "BRL",
  "taxes": {
    "ibs_amount": 1000,
    "cbs_amount": 500,
    "total_tax": 1500,
    "rule_version": "2026-01"
  }
}
```

Campos de identificação como `event_id`, `transaction_id` e `order_id` devem ser preservados em logs e métricas. Dados de cartão, PAN, CVV e credenciais nunca devem fazer parte do evento.

## 5. Política de retry

A política de referência possui três tentativas atrasadas, com backoff fixo por fila: 10 segundos, 60 segundos e 300 segundos. O número máximo é três retries após a tentativa inicial. O intervalo deve ser configurável por ambiente.

| Tentativa | Fila | Atraso | Resultado em nova falha |
|---:|---|---:|---|
| 0 | `payment.processed` | 0 s | Encaminhar para retry 1 |
| 1 | `retry.1` | 10 s | Encaminhar para retry 2 |
| 2 | `retry.2` | 60 s | Encaminhar para retry 3 |
| 3 | `retry.3` | 300 s | Encaminhar para DLQ |

A implementação pode usar headers `x-retry-count`, `x-first-failed-at`, `x-last-error` e `x-original-routing-key`. O erro deve ser truncado para evitar que uma exceção enorme ocupe a mensagem inteira.

## 6. Implementação em Go

### 6.1 Tipos principais

```go
package messaging

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log/slog"
    "time"

    amqp "github.com/rabbitmq/amqp091-go"
)

type RetryableError struct{ Err error }
func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

type PermanentError struct{ Err error }
func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

type PaymentProcessed struct {
    EventID       string    `json:"event_id"`
    EventType     string    `json:"event_type"`
    SchemaVersion int       `json:"schema_version"`
    OccurredAt    time.Time `json:"occurred_at"`
    TransactionID string    `json:"transaction_id"`
    OrderID       string    `json:"order_id"`
    UserID        string    `json:"user_id"`
    Status        string    `json:"status"`
    AmountMinor   int64     `json:"amount_minor"`
    Currency      string    `json:"currency"`
    Taxes         Taxes     `json:"taxes"`
}

type Taxes struct {
    IBSAmount   int64  `json:"ibs_amount"`
    CBSAmount   int64  `json:"cbs_amount"`
    TotalTax    int64  `json:"total_tax"`
    RuleVersion string `json:"rule_version"`
}

type ProcessedEventStore interface {
    AlreadyProcessed(ctx context.Context, eventID string) (bool, error)
    MarkProcessed(ctx context.Context, eventID, consumer string) error
}

type CartService interface {
    ApplyPayment(ctx context.Context, event PaymentProcessed) error
}
```

### 6.2 Configuração da topologia

```go
type Topology struct {
    Exchange       string
    DeadLetterExch string
    MainQueue      string
    RetryQueues    []string
    DLQ            string
    RoutingKey     string
}

func DeclareTopology(ch *amqp.Channel, t Topology) error {
    if err := ch.ExchangeDeclare(t.Exchange, "topic", true, false, false, false, nil); err != nil {
        return fmt.Errorf("declare exchange: %w", err)
    }
    if err := ch.ExchangeDeclare(t.DeadLetterExch, "direct", true, false, false, false, nil); err != nil {
        return fmt.Errorf("declare dlx: %w", err)
    }

    if _, err := ch.QueueDeclare(t.MainQueue, true, false, false, false, amqp.Table{
        "x-dead-letter-exchange":    t.DeadLetterExch,
        "x-dead-letter-routing-key": "payment.processed.dlq",
    }); err != nil {
        return fmt.Errorf("declare main queue: %w", err)
    }
    if err := ch.QueueBind(t.MainQueue, t.RoutingKey, t.Exchange, false, nil); err != nil {
        return fmt.Errorf("bind main queue: %w", err)
    }

    delays := []int32{10_000, 60_000, 300_000}
    for i, queue := range t.RetryQueues {
        if i >= len(delays) {
            return fmt.Errorf("retry queue %q has no configured delay", queue)
        }
        if _, err := ch.QueueDeclare(queue, true, false, false, false, amqp.Table{
            "x-message-ttl":             delays[i],
            "x-dead-letter-exchange":    t.Exchange,
            "x-dead-letter-routing-key": t.RoutingKey,
        }); err != nil {
            return fmt.Errorf("declare retry queue %s: %w", queue, err)
        }
        if err := ch.QueueBind(queue, "payment.retry."+fmt.Sprint(i+1), t.DeadLetterExch, false, nil); err != nil {
            return fmt.Errorf("bind retry queue %s: %w", queue, err)
        }
    }

    if _, err := ch.QueueDeclare(t.DLQ, true, false, false, false, nil); err != nil {
        return fmt.Errorf("declare dlq: %w", err)
    }
    if err := ch.QueueBind(t.DLQ, "payment.processed.dlq", t.DeadLetterExch, false, nil); err != nil {
        return fmt.Errorf("bind dlq: %w", err)
    }
    return nil
}
```

Para produção, a declaração deve ser idempotente e executada por uma etapa de inicialização controlada. Alterações incompatíveis de TTL, argumentos ou bindings devem ser aplicadas por migração de infraestrutura, e não de modo silencioso em cada réplica.

### 6.3 Publicação em retry e DLQ

```go
const (
    headerRetryCount     = "x-retry-count"
    headerOriginalKey    = "x-original-routing-key"
    headerFirstFailedAt  = "x-first-failed-at"
    headerLastError      = "x-last-error"
    maxErrorHeaderBytes  = 512
)

func headerInt(headers amqp.Table, key string) int {
    switch n := headers[key].(type) {
    case int:
        return n
    case int32:
        return int(n)
    case int64:
        return int(n)
    default:
        return 0
    }
}

func cloneHeaders(in amqp.Table) amqp.Table {
    out := make(amqp.Table, len(in)+4)
    for k, v := range in {
        out[k] = v
    }
    return out
}

func truncate(s string) string {
    if len(s) <= maxErrorHeaderBytes {
        return s
    }
    return s[:maxErrorHeaderBytes]
}

func publishRetry(ch *amqp.Channel, msg amqp.Delivery, exchange, routingKey string, err error) error {
    headers := cloneHeaders(msg.Headers)
    headers[headerRetryCount] = headerInt(headers, headerRetryCount) + 1
    headers[headerOriginalKey] = msg.RoutingKey
    headers[headerLastError] = truncate(err.Error())
    if _, ok := headers[headerFirstFailedAt]; !ok {
        headers[headerFirstFailedAt] = time.Now().UTC().Format(time.RFC3339)
    }

    return ch.PublishWithContext(context.Background(), exchange, routingKey, false, false, amqp.Publishing{
        ContentType:  msg.ContentType,
        ContentEncoding: msg.ContentEncoding,
        Body:         msg.Body,
        Headers:      headers,
        DeliveryMode: amqp.Persistent,
        MessageId:    msg.MessageId,
        Timestamp:    time.Now().UTC(),
    })
}

func publishDLQ(ch *amqp.Channel, msg amqp.Delivery, exchange string, err error) error {
    headers := cloneHeaders(msg.Headers)
    headers[headerLastError] = truncate(err.Error())
    headers["x-dead-lettered-at"] = time.Now().UTC().Format(time.RFC3339)
    return ch.PublishWithContext(context.Background(), exchange, "payment.processed.dlq", false, false, amqp.Publishing{
        ContentType:  msg.ContentType,
        Body:         msg.Body,
        Headers:      headers,
        DeliveryMode: amqp.Persistent,
        MessageId:    msg.MessageId,
        Timestamp:    time.Now().UTC(),
    })
}
```

### 6.4 Consumidor idempotente

```go
type Consumer struct {
    ch             *amqp.Channel
    cart           CartService
    processed      ProcessedEventStore
    topology       Topology
    logger         *slog.Logger
    maxRetries     int
    retryRoutes    []string
    consumerName   string
}

func (c *Consumer) Handle(ctx context.Context, msg amqp.Delivery) error {
    var event PaymentProcessed
    if err := json.Unmarshal(msg.Body, &event); err != nil {
        return &PermanentError{Err: fmt.Errorf("decode event: %w", err)}
    }
    if event.EventID == "" || event.EventType != "PaymentProcessed" || event.SchemaVersion != 1 {
        return &PermanentError{Err: errors.New("invalid payment event contract")}
    }

    already, err := c.processed.AlreadyProcessed(ctx, event.EventID)
    if err != nil {
        return &RetryableError{Err: fmt.Errorf("check processed event: %w", err)}
    }
    if already {
        return nil
    }

    if err := c.cart.ApplyPayment(ctx, event); err != nil {
        // O adaptador deve converter erros conhecidos em RetryableError ou PermanentError.
        return err
    }
    if err := c.processed.MarkProcessed(ctx, event.EventID, c.consumerName); err != nil {
        return &RetryableError{Err: fmt.Errorf("mark event processed: %w", err)}
    }
    return nil
}

func (c *Consumer) Run(ctx context.Context, deliveries <-chan amqp.Delivery) {
    for {
        select {
        case <-ctx.Done():
            return
        case msg, ok := <-deliveries:
            if !ok {
                return
            }
            c.processOne(ctx, msg)
        }
    }
}

func (c *Consumer) processOne(ctx context.Context, msg amqp.Delivery) {
    err := c.Handle(ctx, msg)
    if err == nil {
        if ackErr := msg.Ack(false); ackErr != nil {
            c.logger.Error("ack failed", "error", ackErr)
        }
        return
    }

    var permanent *PermanentError
    if errors.As(err, &permanent) {
        c.toDLQ(msg, err)
        return
    }

    retryCount := headerInt(msg.Headers, headerRetryCount)
    if retryCount >= c.maxRetries {
        c.toDLQ(msg, fmt.Errorf("retry limit reached: %w", err))
        return
    }

    route := c.retryRoutes[retryCount]
    if err := publishRetry(c.ch, msg, c.topology.DeadLetterExch, route, err); err != nil {
        // Se o envio ao retry falhar, não confirmar a mensagem original.
        // O broker poderá redeliver; alertas devem detectar esse estado.
        c.logger.Error("publish retry failed", "error", err, "event_id", msg.MessageId)
        _ = msg.Nack(false, true)
        return
    }
    if err := msg.Ack(false); err != nil {
        c.logger.Error("ack after retry publish failed", "error", err)
    }
}

func (c *Consumer) toDLQ(msg amqp.Delivery, cause error) {
    if err := publishDLQ(c.ch, msg, c.topology.DeadLetterExch, cause); err != nil {
        c.logger.Error("publish dlq failed", "error", err, "event_id", msg.MessageId)
        _ = msg.Nack(false, true)
        return
    }
    if err := msg.Ack(false); err != nil {
        c.logger.Error("ack after dlq publish failed", "error", err)
    }
}
```

A confirmação após publicação em retry ou DLQ cria uma janela em que a publicação pode existir e o `Ack` pode falhar. A mensagem poderá ser duplicada, por isso o consumidor e a operação de DLQ devem ser idempotentes. Para garantias mais fortes, use publisher confirms, uma chave de deduplicação operacional e reconciliação periódica.

### 6.5 Publisher confirms

```go
func enableConfirms(ch *amqp.Channel) error {
    return ch.Confirm(false)
}

func publishConfirmed(ch *amqp.Channel, exchange, key string, p amqp.Publishing) error {
    confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
    if err := ch.PublishWithContext(context.Background(), exchange, key, false, false, p); err != nil {
        return err
    }
    confirmation := <-confirms
    if !confirmation.Ack {
        return errors.New("broker rejected published message")
    }
    return nil
}
```

Em uma implementação final, o canal de confirms deve ser criado uma única vez por canal e correlacionado com uma estratégia de publicação apropriada. Não se deve criar um consumidor de confirmações novo a cada mensagem sem controlar concorrência, ordem e encerramento.

## 7. Configuração de prefetch e graceful shutdown

O consumidor deve definir `Qos` para limitar mensagens em voo e impedir que uma réplica acumule mais trabalho do que consegue processar. O encerramento deve cancelar o consumo, aguardar handlers ativos e fechar o canal somente depois que as confirmações necessárias forem emitidas.

```go
if err := ch.Qos(20, 0, false); err != nil {
    return fmt.Errorf("set qos: %w", err)
}

deliveries, err := ch.Consume(
    "payment.processed",
    "cart-payment-consumer",
    false, // autoAck=false
    false,
    false,
    false,
    nil,
)
if err != nil {
    return fmt.Errorf("consume: %w", err)
}
```

## 8. Operação da DLQ

A DLQ exige ownership, retenção, alerta e procedimento de replay. Cada mensagem deve permitir responder: qual evento falhou, qual foi a última exceção, quantas tentativas ocorreram, quando ocorreu a primeira falha e qual serviço deve corrigir o problema.

O replay não deve ser feito diretamente em produção sem validação. O operador deve corrigir a causa, selecionar mensagens por `event_id`, remover ou preservar os headers de retry conforme a política e republicar de forma controlada. Mensagens que falharem novamente devem retornar à DLQ com novo contexto.

Métricas mínimas:

| Métrica | Objetivo |
|---|---|
| `messaging_consumer_processed_total` | Volume processado |
| `messaging_consumer_failures_total` | Falhas por classe |
| `messaging_retry_published_total` | Quantidade de retries |
| `messaging_dlq_published_total` | Entradas na DLQ |
| `messaging_dlq_age_seconds` | Idade da mensagem mais antiga |
| `messaging_processing_duration_seconds` | Latência do handler |

## 9. Testes obrigatórios

Os testes devem verificar que JSON inválido vai diretamente para DLQ; falha transitória entra na fila correta; o terceiro retry excedido vai para DLQ; evento duplicado não reaplica o pagamento; falha de publicação não confirma a mensagem original; e headers são preservados sem expor dados sensíveis.

Também deve haver teste de integração com RabbitMQ efêmero, teste de encerramento gracioso, teste de reconexão e teste de saturação de prefetch. A suíte deve validar que o serviço não executa `Nack(requeue=true)` indefinidamente.

## Referências

[1]: https://manus.im/share/0o55Q3fHjBwBIM7nwxzRKn "Conversa compartilhada — documentação do Payment Service"
[2]: https://www.rabbitmq.com/docs/dlx "RabbitMQ — Dead Letter Exchanges"
[3]: https://www.rabbitmq.com/docs/confirms "RabbitMQ — Consumer Acknowledgements and Publisher Confirms"
[4]: https://github.com/rabbitmq/amqp091-go "RabbitMQ AMQP 0-9-1 Go client"
