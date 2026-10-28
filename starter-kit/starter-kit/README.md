# modkit starter

Go CLI starter for modular DDD and event-driven applications.

## Included Commands

```bash
modkit init
modkit module create <name>
modkit entity create <name> --module <module> [--field name:type]
modkit module list
modkit module add-element <module> <element>
modkit module validate <module>
modkit version
modkit system
```

## Build

```bash
go mod tidy
go build -o bin/modkit .
```

On Windows:

```powershell
go build -o bin/modkit.exe .
.\bin\modkit.exe system
```

## Initialize a Project

From another Go project, run:

```bash
/path/to/modkit init
```

`init` asks for the project name and uses it for `project.name` and the Go
module. Press Enter to use the default name.

For automation, pass the name directly:

```bash
/path/to/modkit init --name payments-api
```

This creates the module `github.com/your-org/payments-api`.

### Project Templates

The default project template is `http`:

```bash
/path/to/modkit init
go run .
```

It generates a Gin HTTP server on `:8080` with `GET /healthz`. The generated
project declares `github.com/gin-gonic/gin` in `go.mod`; run `go mod tidy` after
creating the project to resolve its dependency graph.

Use the `console` template when an HTTP server is not required:

```bash
/path/to/modkit init --template console
```

Available templates are shown by:

```bash
/path/to/modkit init --help
```

The selected project template is stored in `project.template`. The
`defaults.template` setting controls the module template independently.

## Configuration

```yaml
project:
  name: payments-api
  module: github.com/your-org/payments-api
  architecture: modular-ddd
  template: http
modules:
  root: internal
defaults:
  template: console
```

## Create a Module

```bash
/path/to/modkit module create credit-request
```

By default, this creates the module core and domain event contracts. Use
`--create-all` to generate every available element.

```bash
/path/to/modkit module create credit-request --create-all
```

Default structure:

```text
internal/credit-request/
├── application/
│   ├── ports.go
│   └── service.go
├── domain/
│   ├── entity.go
│   ├── errors.go
│   ├── event.go
│   └── repository.go
├── infrastructure/
│   └── repository.go
├── interfaces/
│   └── http/
│       └── handler.go
└── module.go
```

In an HTTP project, module creation also registers the handler in the root
router and generates `cmd/creditrequest_routes.go`. The routes are `POST /credit-request`, `GET /credit-request`, and
`GET|PUT|DELETE /credit-request/:id`.

## Add Elements Incrementally

```bash
/path/to/modkit module add-element credit-request http
/path/to/modkit module add-element credit-request messaging
/path/to/modkit module add-element credit-request outbox
/path/to/modkit module add-element credit-request threading
/path/to/modkit module add-element credit-request observability
```

Use `modkit module add-element --help` to see the available elements.

With `--create-all`, the optional files include:

```text
infrastructure/event_publisher.go
infrastructure/outbox.go
infrastructure/worker_group.go
interfaces/consumer/handler.go
interfaces/observability/hooks.go
```

## Module Element Guide

### Domain

The domain contains entities, domain events, repository contracts, and domain
errors. It should not import databases, brokers, HTTP packages, or frameworks.

Entities record events with `Record` and expose them with `PullEvents`:

```go
entity := domain.NewCreditRequest("request-1")
entity.Record(domain.NewCreditRequestCreated(entity.ID))
events := entity.PullEvents()
```

`event.go` defines the event name, aggregate ID, and UTC timestamp. Add a new
event when a business state transition matters outside the aggregate.

### Application

`service.go` coordinates the use case: create the entity, record its event,
save it through the repository, and publish pending events.

```go
service := application.NewServiceWithPublisher(repository, publisher)
err := service.Create(ctx, "request-1")
```

`ports.go` defines the `EventPublisher` port and a `NoopPublisher` for projects
that have not selected a broker yet.

### Infrastructure

`repository.go` implements CRUD with GORM and the shared PostgreSQL connection.

`event_publisher.go` connects the application publisher port to Kafka, NATS,
RabbitMQ, SNS/SQS, or another broker.

`outbox.go` defines `Append`, `Pending`, and `MarkPublished`. A production
outbox should persist the event in the same transaction as the aggregate and
dispatch it asynchronously with retries and idempotency.

### Threading

`worker_group.go` coordinates independent goroutines under one context. The
first worker error cancels the shared context; the group waits for all workers
and returns that error.

```go
workers := infrastructure.NewWorkerGroup(
    func(ctx context.Context) error {
        for {
            select {
            case <-ctx.Done():
                return ctx.Err()
            default:
                // process one unit of work
            }
        }
    },
)
err := workers.Run(ctx)
```

The helper does not provide a queue, scheduler, retry policy, or shared-state
synchronization. Use channels or the appropriate `sync` primitive for shared
state.

### Interfaces

`interfaces/http/handler.go` defines the Gin HTTP boundary and `RegisterRoutes`.
It maps requests to application use cases and results to HTTP responses.

`interfaces/consumer/handler.go` defines the boundary for incoming broker
events. Deserialize the message and acknowledge it only after successful
processing.

`interfaces/observability/hooks.go` exposes hooks for command start/finish and
event publication. Connect them to logs, metrics, traces, and correlation IDs.

### Composition Root

`module.go` wires the repository and application service without forcing a
specific database or broker into the domain.

## Validate a Module

```bash
/path/to/modkit module validate credit-request
```

Validation checks that the generated core files exist. It does not check
runtime connectivity, migrations, broker configuration, or business invariants.

## Recommended Extensions

- persistence and messaging adapters: PostgreSQL, Redis, Kafka, NATS, RabbitMQ, SNS/SQS
- transactional outbox with dispatcher
- retries, idempotency, dead-letter queues, and message deduplication
- event schema versioning and contract tests
- observability with logs, metrics, tracing, and correlation IDs
- external templates, project presets, releases, and container images

For the Spanish guide, see [LEEME.md](LEEME.md).


## PostgreSQL and additional entities

New HTTP and console projects include GORM with PostgreSQL. Set `DB_DSN` to
an existing database connection string before running the application:

```bash
modkit entity create customer --module sales --field name:string --field active:bool
go mod tidy
export DB_DSN='host=localhost port=5433 user=postgres password=example dbname=postgres sslmode=disable'
go run .
```

Create `sales` first with `modkit module create sales`. Additional aggregates live
under `internal/sales/entities/customer/` and expose POST/GET at `/sales/customer`
and GET/PUT/DELETE at `/sales/customer/:id`. POST takes a JSON body including a
client-supplied string ID. PUT replaces all fields, including zero values.
Supported repeatable fields: string, bool, int, int64, uint, uint64, float64.
Use hyphen-separated field names for PascalCase Go and snake_case JSON names.
Existing files are protected unless `--force` is supplied.

HTTP startup opens one shared database and calls each module's AutoMigrate.
Console consumers explicitly call `module.New(db)`, returning `(*Module, error)`.
Tables are namespaced by module/entity. Lists use limit/offset pagination (defaults 20/0, maximum limit 100).
Responses use 400 for invalid input, 404 for missing records, and 409 for duplicates.
The generated connection follows the [GORM PostgreSQL setup](https://gorm.io/docs/connecting_to_the_database.html#PostgreSQL).

Existing projects need to update their composition root, database setup and
dependencies to use new generated routes. The earlier query-string POST is
replaced by JSON; the application API still supports `Create(ctx, id)`.

Run `go test ./...` for generator checks. Set `MODKIT_TEST_POSTGRES_DSN` to run
the real PostgreSQL CRUD test; it uses a temporary schema in a rolled-back
transaction. See LEEME.md for the complete usage examples.


## List pagination / Paginacion de listados

```bash
curl 'localhost:8080/sales/customer?limit=10&offset=10'
```

```json
{"items":[],"limit":10,"offset":10,"total":0}
```

All generated module and entity CRUD lists use limit (default 20, range 1–100)
and offset (default 0, non-negative). Offset is the number of records to skip
and need not be a multiple of limit. Invalid, empty, or out-of-range values return
400. Offsets beyond the total return empty items with the total preserved. Results order by ID.
GORM applies COUNT, LIMIT, and OFFSET. Count and data queries are separate;
concurrent writes may change results between queries.

The service signature is `List(ctx, domain.Pagination{Limit: 20, Offset: 0})`.
HTTP responses now wrap items and metadata instead of returning a bare array.
Existing consumers and generated modules require adaptation; --force overwrites files.
