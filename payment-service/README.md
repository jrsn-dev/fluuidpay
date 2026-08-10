# Fluuid Pay — Payment Service

Microsserviço de pagamentos em Go, baseado em Clean Architecture, com tokenização de cartão, idempotência, tributação IBS/CBS, mensageria assíncrona (RabbitMQ) e DLQ.

## Arquitetura

```
cmd/api/main.go          → Entrypoint + DI
internal/domain/         → Entidades, VOs, interfaces (zero deps de infra)
internal/usecase/        → Casos de uso (orquestração)
internal/adapter/http/   → Handlers HTTP (Chi)
internal/adapter/gateway/    → Adaptador de gateway de pagamento
internal/adapter/tax/        → Calculadora IBS/CBS
internal/adapter/messaging/  → Publisher/Consumer RabbitMQ
internal/adapter/repository/ → PostgreSQL + Redis repos
internal/platform/       → Config, DB, Redis, RabbitMQ, Telemetria
```

## Quick Start

### Pré-requisitos

- Go 1.26+
- Docker e Docker Compose

### Subir infraestrutura

```bash
make docker-up
```

Isso inicia PostgreSQL, Redis e RabbitMQ com health checks.

### Rodar migrações

```bash
make migrate-up
```

### Iniciar o serviço

```bash
make dev
```

O serviço estará em `http://localhost:8080`.

### Executar testes

```bash
make test
```

## API

A especificação OpenAPI está em `api/openapi.yaml`.

### Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/v1/payments` | Criar pagamento (requer `Idempotency-Key`) |
| GET | `/v1/payments/{id}` | Consultar pagamento |
| POST | `/v1/payments/{id}/cancel` | Cancelar pagamento |
| POST | `/v1/webhooks/payment-gateway` | Receber webhook do gateway |
| GET | `/health` | Liveness check |
| GET | `/ready` | Readiness check |

### Exemplo

```bash
curl -X POST http://localhost:8080/v1/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "user_id": "user-123",
    "order_id": "order-456",
    "amount_minor": 19990,
    "currency": "BRL",
    "card_token": "tok_provider_abc",
    "destination": {
      "country_code": "BR",
      "state_code": "SP"
    }
  }'
```

## Decisões de Arquitetura

- **Banco:** PostgreSQL como fonte de verdade
- **Cache/Idempotência:** Redis (hot) + PostgreSQL (cold)
- **Mensageria:** RabbitMQ com Outbox transacional
- **Valores monetários:** `int64` em menor unidade (centavos)
- **Segurança:** Tokenização estrita, sem PAN/CVV em nenhuma camada

## Documentação

- [SDD Completo](docs/sdd.md)
- [Plano de Melhorias](docs/melhorias.md)
- [Retries & DLQ](docs/retries_dlq.md)
- [OpenAPI](api/openapi.yaml)
