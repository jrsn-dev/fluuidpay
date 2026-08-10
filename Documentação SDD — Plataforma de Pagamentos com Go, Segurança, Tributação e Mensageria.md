# Documentação SDD — Plataforma de Pagamentos com Go, Segurança, Tributação e Mensageria

**Autor:** Manus AI  
**Fonte principal:** conversa compartilhada pelo usuário [1]  
**Data de consolidação:** 10 de agosto de 2026  
**Versão:** 1.0.0 — consolidação analítica

> Este documento reconstrói e organiza a conversa compartilhada em uma especificação orientada por SDD — *Specification-Driven Development*. Ele distingue claramente o que foi definido na conversa, o que foi implementado apenas parcialmente, o que ficou pendente e quais passos devem ser executados para transformar a especificação em um sistema produtivo.

## 1. Objetivo e escopo

O objetivo é definir um microsserviço de pagamentos em Go, baseado em Clean Architecture, capaz de processar cobranças usando tokens de cartão fornecidos por gateways certificados, impedir cobranças duplicadas por meio de idempotência, calcular tributos de forma desacoplada e comunicar-se de maneira assíncrona com o microsserviço de Carrinho.

O escopo consolidado inclui o domínio de pagamentos, o contrato HTTP, a integração com gateway externo, a persistência de idempotência em Redis, o motor tributário IBS/CBS, a comunicação assíncrona entre Pagamentos e Carrinho, mecanismos de retry e DLQ, testes, observabilidade, segurança, implantação e critérios de aceite.

A conversa não apresenta uma implementação completa e verificável de todos esses componentes. O conteúdo disponível mostra a definição arquitetural, parte dos modelos em Go, uma implementação iniciada do caso de uso e a geração de um PDF; a solicitação posterior sobre RabbitMQ/Kafka foi interrompida quando a sessão atingiu o limite de créditos [1]. Portanto, esta documentação também funciona como **baseline de continuidade**.

## 2. Resumo executivo da solução

A solução proposta recebe uma solicitação de pagamento contendo identificador do usuário, valor, moeda, token seguro do cartão e chave de idempotência. O caso de uso valida a requisição, consulta o Redis para determinar se a mesma chave já foi processada, calcula os tributos conforme o destino da transação, encaminha a cobrança ao gateway e persiste ou devolve o resultado de maneira idempotente.

Após o resultado da cobrança, o serviço publica um evento de domínio para que o Carrinho atualize seu estado. A publicação deve ser feita por meio de um padrão transacional, preferencialmente Outbox, para evitar a inconsistência entre o banco de dados e o broker. O consumidor do Carrinho deve ser idempotente, possuir retries controlados e enviar mensagens irrecuperáveis para uma DLQ.

| Elemento | Decisão consolidada |
|---|---|
| Linguagem | Go |
| Organização | Clean Architecture e princípios SOLID |
| Cartões | Somente tokens de gateway; não aceitar PAN ou CVV bruto |
| Idempotência | Header `Idempotency-Key`, com registro no Redis e TTL de 24 horas |
| Tributação | Motor desacoplado, calculando IBS/CBS por destino e contexto da transação |
| Integração financeira | Interface abstrata de gateway externo |
| Integração entre serviços | Mensageria assíncrona entre Pagamentos e Carrinho |
| Resiliência | Retry com backoff, DLQ e consumidores idempotentes |
| Documentação de API | Swagger/OpenAPI |
| Situação ao final da conversa | Arquitetura definida; implementação e mensageria incompletas na evidência disponível |

## 3. Contexto funcional

### 3.1 Atores e sistemas

O **cliente ou checkout** inicia o pagamento. O **Payment Service** valida e orquestra a operação. O **gateway de pagamentos** autoriza ou rejeita a cobrança. O **Redis** mantém o estado de idempotência por tempo determinado. O **motor tributário** calcula os tributos aplicáveis. O **broker de mensagens** transporta eventos. O **Carrinho** reage ao evento de pagamento aprovado, rejeitado ou pendente.

### 3.2 Fluxo principal

1. O checkout envia `POST /v1/payments` com `Idempotency-Key` único.
2. O serviço valida campos, moeda, valor, token e destino tributário.
3. O serviço consulta o Redis.
4. Se a chave já possui uma resposta final, a resposta anterior é devolvida sem nova cobrança.
5. Se a chave está em processamento, o serviço responde com estado de processamento ou conflito, conforme a política escolhida.
6. O motor tributário calcula IBS, CBS e total de tributos.
7. O gateway recebe somente o token seguro e os dados necessários à cobrança.
8. O serviço grava o resultado e publica `PaymentProcessed` por Outbox/broker.
9. O Carrinho consome o evento e atualiza o pedido.
10. Falhas transitórias são reprocessadas; falhas permanentes seguem para DLQ.

### 3.3 Estados do pagamento

| Estado | Significado | Ação do Carrinho |
|---|---|---|
| `PENDING` | O gateway ainda não confirmou o resultado | Manter pedido aguardando confirmação |
| `APPROVED` | Cobrança autorizada | Confirmar pedido e reservar/baixar estoque |
| `REJECTED` | Cobrança negada ou inválida | Liberar reserva e informar falha |
| `CANCELLED` | Pagamento cancelado após aprovação ou antes da captura | Reverter efeitos do pedido |
| `REFUNDED` | Valor devolvido total ou parcialmente | Registrar devolução e ajustar pedido |

## 4. Especificação SDD

### 4.1 Princípio de trabalho

A implementação deve ocorrer a partir de especificações versionadas e verificáveis. Cada requisito precisa possuir uma decisão técnica, um contrato, um teste e um critério de aceite. Nenhuma regra financeira ou tributária deve existir somente em comentário, configuração informal ou lógica espalhada entre handlers.

O ciclo SDD adotado é: **especificar → modelar → implementar → verificar → registrar progressão → evoluir**.

### 4.2 Requisitos funcionais

| ID | Requisito | Critério de aceite |
|---|---|---|
| RF-001 | Criar pagamento | Uma requisição válida retorna identificador e estado do pagamento |
| RF-002 | Aplicar idempotência | Repetições da mesma chave não geram nova cobrança |
| RF-003 | Calcular tributos | A resposta informa IBS, CBS e total calculados pelo motor configurado |
| RF-004 | Integrar gateway | O serviço usa uma interface, sem acoplar o domínio a um fornecedor |
| RF-005 | Publicar evento | Um resultado persistido gera evento consumível pelo Carrinho |
| RF-006 | Reprocessar falhas | Mensagens transitórias são retentadas conforme política definida |
| RF-007 | Isolar mensagens inválidas | Mensagens que excedem retries ou são permanentemente inválidas vão para DLQ |
| RF-008 | Consultar pagamento | Deve existir endpoint de consulta por `transaction_id` |
| RF-009 | Registrar auditoria | Mudanças relevantes geram trilha auditável sem dados sensíveis de cartão |

### 4.3 Requisitos não funcionais

| ID | Requisito | Diretriz |
|---|---|---|
| RNF-001 | Segurança | Nunca armazenar nem registrar PAN ou CVV |
| RNF-002 | Disponibilidade | Isolar indisponibilidade do gateway e do broker por timeouts e circuit breaker |
| RNF-003 | Consistência | Evitar cobrança duplicada e publicação perdida por Outbox/idempotência |
| RNF-004 | Observabilidade | Logs estruturados, métricas e tracing distribuído |
| RNF-005 | Testabilidade | Domínio e casos de uso testáveis sem infraestrutura real |
| RNF-006 | Evolução fiscal | Alíquotas e regras devem ser versionadas e substituíveis |
| RNF-007 | Privacidade | Minimizar dados pessoais, controlar acesso e definir retenção |

## 5. Arquitetura lógica

### 5.1 Camadas

A **camada de domínio** contém entidades, objetos de valor, estados, interfaces e regras invariantes. Ela não conhece HTTP, Redis, broker, SDK de gateway ou banco de dados.

A **camada de aplicação** implementa os casos de uso, coordena idempotência, cálculo tributário, cobrança, persistência e publicação de eventos. Ela depende de interfaces do domínio.

A **camada de adaptadores** converte protocolos externos para os contratos internos. Nela ficam handlers HTTP, consumidores de mensagens, repositórios, adaptadores Redis, gateway e cálculo tributário.

A **camada de infraestrutura** configura servidores, clientes, conexões, migrações, métricas, tracing, secrets e composição da aplicação.

### 5.2 Estrutura sugerida

```text
payment-service/
├── cmd/api/main.go
├── internal/domain/
│   ├── payment.go
│   ├── tax.go
│   └── events.go
├── internal/usecase/
│   └── payment_usecase.go
├── internal/adapter/http/
│   ├── handler.go
│   ├── request.go
│   └── response.go
├── internal/adapter/messaging/
│   ├── publisher.go
│   └── consumer.go
├── internal/adapter/gateway/
│   └── provider.go
├── internal/adapter/tax/
│   └── calculator.go
├── internal/adapter/repository/
│   ├── payment_repository.go
│   └── outbox_repository.go
├── internal/platform/
│   ├── redis.go
│   ├── database.go
│   ├── telemetry.go
│   └── config.go
├── api/openapi.yaml
├── migrations/
├── docs/
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

## 6. Modelo de domínio

A conversa definiu os tipos `TaxDetails`, `PaymentRequest`, `PaymentResponse`, `PaymentGateway` e `TaxCalculator` [1]. A ideia deve ser preservada, mas o modelo precisa ser endurecido para produção.

```go
package domain

import (
    "context"
    "time"
)

type PaymentStatus string

const (
    PaymentPending   PaymentStatus = "PENDING"
    PaymentApproved  PaymentStatus = "APPROVED"
    PaymentRejected  PaymentStatus = "REJECTED"
    PaymentCancelled PaymentStatus = "CANCELLED"
    PaymentRefunded  PaymentStatus = "REFUNDED"
)

type TaxDetails struct {
    IBSAmount  int64  `json:"ibs_amount"`
    CBSAmount  int64  `json:"cbs_amount"`
    TotalTax   int64  `json:"total_tax"`
    Currency   string `json:"currency"`
    RuleVersion string `json:"rule_version"`
}

type PaymentRequest struct {
    UserID         string `json:"user_id"`
    Amount         int64  `json:"amount_minor"`
    Currency       string `json:"currency"`
    CardToken      string `json:"card_token"`
    IdempotencyKey string `json:"-"`
    StateCode      string `json:"state_code"`
    OrderID        string `json:"order_id"`
}

type PaymentResponse struct {
    TransactionID string        `json:"transaction_id"`
    Status        PaymentStatus `json:"status"`
    Taxes         TaxDetails    `json:"taxes"`
    CreatedAt     time.Time     `json:"created_at"`
}

type PaymentGateway interface {
    ProcessCharge(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
}

type TaxCalculator interface {
    CalculateTaxes(ctx context.Context, amount int64, stateCode, currency string) (TaxDetails, error)
}

type IdempotencyStore interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Reserve(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
    Complete(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
```

O uso de `int64` para valores monetários é uma recomendação de robustez: o valor deve ser armazenado na menor unidade da moeda, evitando ambiguidades de ponto flutuante. A regra precisa ser confirmada conforme as moedas suportadas.

## 7. Caso de uso e idempotência

A implementação iniciada na conversa mostra a intenção de consultar o Redis antes de chamar o gateway, mas o trecho disponível está truncado em `idempote` [1]. O comportamento completo deve seguir os passos abaixo.

1. Validar a chave e os campos de entrada.
2. Calcular um hash canônico do payload, excluindo a própria chave.
3. Tentar reservar a chave no Redis com operação atômica `SET NX` e TTL de 24 horas.
4. Se a chave já existir, comparar o hash do payload. Payload diferente deve resultar em erro de conflito; payload igual deve devolver a resposta armazenada ou informar que a operação ainda está em processamento.
5. Executar o cálculo tributário.
6. Chamar o gateway com timeout e contexto cancelável.
7. Persistir o pagamento e a resposta idempotente.
8. Criar o evento Outbox na mesma transação da persistência do pagamento.
9. Marcar a chave como concluída.
10. Devolver a resposta ao cliente.

O Redis não deve ser tratado como a única fonte permanente da transação. O pagamento e o registro de negócio devem permanecer em banco durável; o Redis deve acelerar a deduplicação e controlar a janela de repetição.

## 8. API HTTP e OpenAPI

### 8.1 Criação de pagamento

```http
POST /v1/payments
Idempotency-Key: 8a2c4c9b-...
Content-Type: application/json
Authorization: Bearer <token>
```

```json
{
  "user_id": "user-123",
  "order_id": "order-456",
  "amount_minor": 19990,
  "currency": "BRL",
  "card_token": "tok_provider_abc",
  "state_code": "SP"
}
```

Resposta esperada:

```json
{
  "transaction_id": "txn-789",
  "status": "APPROVED",
  "taxes": {
    "ibs_amount": 0,
    "cbs_amount": 0,
    "total_tax": 0,
    "currency": "BRL",
    "rule_version": "2026-01"
  },
  "created_at": "2026-08-10T14:00:00Z"
}
```

### 8.2 Códigos HTTP

| Situação | Código |
|---|---:|
| Pagamento aprovado ou criado | `201` |
| Solicitação repetida com resposta existente | `200` |
| Operação ainda em processamento | `202` |
| Payload inválido | `400` |
| Chave ausente | `400` |
| Mesma chave com payload diferente | `409` |
| Não autenticado | `401` |
| Gateway indisponível após política de retry | `503` |
| Erro interno | `500` |

O arquivo `api/openapi.yaml` deve ser escrito antes da implementação dos handlers. A especificação deve conter esquemas, exemplos, erros, headers, autenticação e regras de idempotência.

## 9. Tributação e conformidade

O motor tributário deve ser uma dependência abstrata do caso de uso. As alíquotas, exceções, vigências, regras de destino e arredondamentos não devem ser codificados diretamente no handler HTTP ou no gateway.

A implementação deve suportar uma versão de regra, data de vigência, jurisdição, tipo de produto, regime aplicável e origem dos dados. Cada cálculo deve produzir um detalhamento auditável, sem permitir que o cliente altere unilateralmente a alíquota.

| Campo mínimo | Finalidade |
|---|---|
| `state_code` | Determinar o destino tributário informado |
| `rule_version` | Reproduzir o cálculo no futuro |
| `ibs_amount` | Registrar o componente IBS |
| `cbs_amount` | Registrar o componente CBS |
| `total_tax` | Consolidar o valor tributário |
| `calculated_at` | Determinar o instante do cálculo |

A conversa cita IBS/CBS como requisito de flexibilidade fiscal [1]. A parametrização efetiva deve ser revisada com especialistas tributários e com a legislação aplicável antes da entrada em produção; esta documentação não substitui validação jurídica, fiscal ou contábil.

## 10. Mensageria entre Pagamentos e Carrinho

A conversa solicitou uma comparação e implementação com RabbitMQ ou Kafka, porém essa etapa foi interrompida pelo esgotamento de créditos [1]. A decisão deve ser formalizada antes do desenvolvimento do consumidor.

| Critério | RabbitMQ | Kafka |
|---|---|---|
| Modelo predominante | Filas e roteamento por exchanges | Log distribuído e partições |
| Caso mais natural | Comandos/eventos operacionais e entrega por fila | Alto volume, retenção e replay |
| Retenção | Normalmente orientada ao consumo e configuração da fila | Parte central do modelo de consumo |
| Complexidade operacional | Geralmente menor para um fluxo simples | Maior, especialmente com particionamento |
| Recomendação inicial | Preferível se o objetivo é integração direta Pagamentos–Carrinho | Preferível se houver necessidade de replay, analytics ou múltiplos consumidores |

Para o cenário descrito, a recomendação inicial é RabbitMQ quando o fluxo for predominantemente transacional e baseado em entrega de mensagens. Kafka deve ser selecionado se a plataforma exigir retenção prolongada, reprocessamento por offset, vários consumidores independentes ou alto volume. A escolha final precisa considerar volume, SLO, equipe, operação e requisitos de replay.

### 10.1 Evento de pagamento

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
    "ibs_amount": 0,
    "cbs_amount": 0,
    "total_tax": 0,
    "rule_version": "2026-01"
  }
}
```

### 10.2 Retry, DLQ e idempotência do consumidor

O produtor deve publicar com confirmação do broker e persistência adequada. O consumidor deve validar o schema, registrar `event_id` processado e executar a atualização do Carrinho de maneira idempotente. Falhas transitórias devem utilizar backoff exponencial com limite de tentativas. Falhas de schema, contrato ou dados devem ser classificadas como permanentes e direcionadas para DLQ.

O fluxo recomendado é:

1. Consumir a mensagem.
2. Validar envelope e versão do schema.
3. Verificar se `event_id` já foi aplicado.
4. Aplicar a transição de estado do Carrinho.
5. Registrar o evento como processado.
6. Confirmar a mensagem.
7. Em falha transitória, reencaminhar para retry.
8. Após o limite ou em falha permanente, publicar na DLQ com motivo e contexto técnico.

## 11. Segurança e PCI-DSS

A decisão central da conversa é a tokenização estrita: o backend não deve receber, armazenar ou registrar PAN e CVV brutos; deve receber tokens emitidos por gateways certificados [1]. Essa regra precisa ser reforçada com validação de formato, mascaramento de logs, segregação de segredos, TLS, controle de acesso, rotação de credenciais e revisão de dependências.

Nenhuma exceção de debug deve imprimir o corpo completo da requisição. Logs devem conter `transaction_id`, `order_id` e correlação, mas não token integral, credenciais, CVV, PAN ou resposta sensível do gateway.

| Controle | Implementação recomendada |
|---|---|
| Segredos | Secret manager ou variáveis injetadas em runtime |
| Transporte | TLS entre cliente, serviço, gateway, Redis, banco e broker |
| Autorização | JWT/OAuth2 ou mecanismo corporativo equivalente |
| Logs | Redação automática de campos sensíveis |
| Dependências | SCA, atualização e verificação de vulnerabilidades |
| Auditoria | Registro imutável de decisões e transições sem dados de cartão |
| Rede | Segmentação e egress controlado para gateway |

## 12. Persistência e consistência

Embora a conversa destaque Redis, um serviço de pagamentos produtivo precisa de armazenamento durável para transações, estados, valores, moeda, resultado do gateway, tributos, auditoria e Outbox. O esquema exato não foi definido na conversa e deve ser aprovado como uma decisão de arquitetura.

Tabelas ou agregados mínimos:

| Estrutura | Conteúdo |
|---|---|
| `payments` | Identidade, pedido, valor, moeda, estado e timestamps |
| `payment_attempts` | Tentativas, gateway, código externo e resultado sanitizado |
| `idempotency_records` | Chave, hash do payload, estado e resposta serializada |
| `outbox_events` | Evento, payload, status de publicação e tentativas |
| `processed_events` | `event_id`, consumidor e data de processamento |
| `tax_calculations` | Valores, versão da regra e jurisdição |

O padrão Outbox é recomendado porque reduz o risco de uma cobrança persistida sem evento ou de um evento publicado sem o estado correspondente. O publicador deve ser reexecutável e marcar o evento como publicado somente após confirmação do broker.

## 13. Observabilidade e operação

A solução deve definir métricas antes de ser considerada pronta. Métricas essenciais incluem latência do endpoint, taxa de aprovação, rejeições por motivo, erros do gateway, colisões de idempotência, tamanho da Outbox, idade da mensagem mais antiga, retries, mensagens na DLQ e falhas no cálculo tributário.

Cada requisição deve possuir um `correlation_id`. O mesmo identificador, além do `transaction_id`, deve acompanhar logs, traces, chamadas ao gateway e eventos. Alertas devem ser ligados a SLOs, não apenas a exceções isoladas.

## 14. Estratégia de testes

A estratégia deve combinar testes unitários, de contrato, integração e resiliência.

| Camada | Testes obrigatórios |
|---|---|
| Domínio | Transições de estado, valores inválidos e invariantes |
| Caso de uso | Idempotência, gateway, impostos, timeout e falhas |
| Redis | `SET NX`, TTL, colisão de payload e resposta repetida |
| Gateway | Adaptação de sucesso, rejeição, timeout e erro transitório |
| HTTP | Schema, autenticação, headers e códigos de erro |
| Broker | Publicação, consumo, confirmação e DLQ |
| Outbox | Persistência atômica e reprocessamento seguro |
| Segurança | Redação de logs, ausência de PAN/CVV e autorização |
| Contrato | Compatibilidade do evento entre Pagamentos e Carrinho |
| Carga | Latência, concorrência e comportamento sob indisponibilidade |

Critérios mínimos de aceite incluem: duas requisições concorrentes com a mesma chave resultam em uma única cobrança; o mesmo payload repetido devolve a mesma resposta; payload divergente com a mesma chave retorna `409`; nenhum teste encontra PAN/CVV em logs ou banco; o evento é publicado após persistência; o consumidor pode receber o mesmo evento mais de uma vez sem duplicar efeitos; e uma mensagem inválida termina na DLQ com motivo rastreável.

## 15. Plano de implementação passo a passo

### Fase 0 — Decisões pendentes

Registrar ADRs para banco, broker, gateway, estratégia de captura/autorização, política de retry, duração da idempotência, esquema de eventos e fonte das regras fiscais. Sem essas decisões, a equipe deve evitar codificar contratos irreversíveis.

### Fase 1 — Especificação

Criar `README.md`, requisitos funcionais, requisitos não funcionais, modelo de ameaças, OpenAPI, schemas de eventos e critérios de aceite. Cada item deve receber identificador estável.

### Fase 2 — Domínio

Implementar entidades, estados, validações, objetos de valor monetário, interfaces e erros tipados. Escrever testes unitários independentes de infraestrutura.

### Fase 3 — Caso de uso

Implementar o fluxo de pagamento com reserva idempotente, cálculo de tributos, chamada ao gateway, persistência e criação da Outbox. Definir claramente o comportamento em falhas e concorrência.

### Fase 4 — Adaptadores

Implementar handler HTTP, adaptador Redis, repositórios, adaptador do gateway, calculador tributário e publicador. Adicionar timeouts, métricas, logs estruturados e tracing.

### Fase 5 — Mensageria

Escolher RabbitMQ ou Kafka, criar o contrato `PaymentProcessed`, implementar publicação confirmada, consumidor do Carrinho, tabela de eventos processados, retries e DLQ.

### Fase 6 — Verificação

Executar lint, testes unitários, integração com dependências efêmeras, testes de contrato, testes de concorrência e análise de segurança. Gerar relatório versionado.

### Fase 7 — Implantação controlada

Configurar ambientes, migrações, secrets, health checks, readiness, observabilidade, rollback e testes de fumaça. Liberar inicialmente com feature flag ou percentual controlado.

### Fase 8 — Operação e evolução

Acompanhar métricas, erros, DLQ, reconciliação com gateway e divergências tributárias. Revisar ADRs, schemas e regras fiscais por versão.

## 16. Documentação de progressão da conversa

| Ordem | Etapa observada | Resultado | Situação |
|---:|---|---|---|
| 1 | Definição do tema | Sistema de API/pagamentos em Go com Swagger | Concluída |
| 2 | Segurança e tributação | Tokenização, PCI-DSS, idempotência Redis e IBS/CBS | Definida em nível arquitetural |
| 3 | Clean Architecture | Domínio, gateway e calculador tributário como interfaces | Parcialmente documentada |
| 4 | Implementação Go | Modelos e início do caso de uso | Incompleta no trecho disponível |
| 5 | Documento PDF | Conversão do Markdown para PDF concluída | Concluída na sessão original |
| 6 | Mensageria | Solicitação de RabbitMQ/Kafka, retries e DLQ | Não concluída |
| 7 | Encerramento | Sessão interrompida por esgotamento de créditos | Bloqueio operacional registrado |

A progressão deve ser retomada a partir da Fase 0 do plano de implementação. O próximo incremento lógico não é simplesmente gerar mais código: é fechar as decisões pendentes, completar o contrato OpenAPI/eventos e, então, implementar o caso de uso com testes de concorrência e integração.

## 17. Lacunas e riscos identificados

A conversa não informa qual banco de dados será usado, qual gateway foi escolhido, se a cobrança é autorização ou captura imediata, quais são os SLAs, qual broker será adotado, qual mecanismo de autenticação será utilizado, quais são as regras tributárias oficiais, como ocorrerá reconciliação financeira ou qual é a política de retenção de dados.

Também não há evidência disponível de um código compilável completo, migrações, `go.mod`, testes, OpenAPI final, configuração de broker, consumidor do Carrinho, DLQ, dashboards ou pipeline CI/CD. O PDF mencionado não estava disponível no diretório desta sessão para validação independente.

| Risco | Impacto | Mitigação |
|---|---|---|
| Cobrança duplicada | Financeiro e reputacional | Reserva atômica, estado persistente e testes concorrentes |
| Divergência entre pagamento e Carrinho | Operacional | Outbox, consumidor idempotente e reconciliação |
| Regra fiscal incorreta | Fiscal e financeiro | Versionamento, aprovação especializada e testes por vigência |
| Vazamento de dados de cartão | Crítico | Tokenização, redaction, escopo PCI e revisão de segurança |
| Gateway indisponível | Receita e experiência | Timeouts, circuit breaker, retry e estado `PENDING` |
| DLQ sem operação | Mensagens perdidas | Alertas, runbook, replay controlado e ownership |
| Contrato incompatível | Falha entre serviços | Schema versionado e testes de contrato |

## 18. Definição de pronto

O serviço será considerado pronto quando a especificação estiver versionada, os contratos HTTP e assíncronos forem aprovados, o código compilar, os testes automatizados cobrirem os critérios críticos, a idempotência estiver validada sob concorrência, os dados sensíveis estiverem protegidos, a Outbox e a DLQ forem operacionais, a observabilidade estiver disponível e houver runbook para falhas do gateway, broker, Redis, banco e regras tributárias.

A aprovação final deve ser registrada em uma matriz de rastreabilidade ligando cada requisito a pelo menos um teste, uma implementação e uma evidência de execução.

## 19. Próximos passos recomendados

O próximo passo imediato é transformar as lacunas em ADRs e confirmar quatro decisões: banco durável, gateway, broker e fonte/versionamento das regras tributárias. Em seguida, deve-se completar o OpenAPI e o schema `PaymentProcessed`, implementar o caso de uso com uma abstração de `IdempotencyStore`, adicionar Outbox e produzir a primeira suíte de testes concorrentes.

Depois disso, a equipe deve implementar o adaptador de mensageria escolhido, o consumidor idempotente do Carrinho e a DLQ, validar os critérios de aceite em ambiente de integração e somente então gerar a versão final do PDF e dos artefatos de entrega.

## Referências

[1]: https://manus.im/share/0o55Q3fHjBwBIM7nwxzRKn "Conversa compartilhada — Documentação Técnica para Sistema de API com Golang e Swagger"
