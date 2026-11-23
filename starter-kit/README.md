# modkit starter

Go CLI for modular monoliths with pragmatic DDD, PostgreSQL, GORM, and Gin.
Business modules share domain, application, infrastructure, and HTTP layers.
Creating a module does not create an implicit entity.

See the [user guide](docs/user-guide.md) for complete usage instructions.
The Spanish guide remains available in [LEEME.md](LEEME.md).

## Build and start

```bash
go build -o bin/modkit .
go test ./...
modkit init --name loan-origination
modkit module create origination
modkit entity create application --module origination --field amount:int64
modkit entity create applicant --module origination --field name:string --no-crud
go mod tidy
export DB_DSN='host=localhost port=5433 user=postgres password=example dbname=postgres sslmode=disable'
go run ./tools/migrate up
go run .
```

HTTP is the default template (:8080, GET /healthz). Use init --template console
for a console application. PostgreSQL setup lives in
internal/platform/database/database.go. DB_DSN is required.

## Shared module layers

```text
origination/
├── .modkit.json
├── module.go
├── module_generated.go
├── domain/
│   ├── application.go
│   ├── application_events.go
│   ├── application_repository.go
│   ├── applicant.go
│   ├── applicant_events.go
│   ├── events.go
│   ├── errors.go
│   └── pagination.go
├── application/
│   ├── application_service.go
│   └── ports.go
├── infrastructure/
│   └── application_repository.go
└── http/
    └── application_handler.go
```

New entities add named files to these shared packages. Shared contracts are
generated once. module.go is for custom integration code; module_generated.go
owns generated composition and must not be edited. Commit .modkit.json, which
tracks entity definitions and generation policy. Custom files are preserved.

CRUD modules expose New(db) returning (*Module, error), build named services without modifying the database schema. HTTP modules expose RegisterRoutes and are
registered once in cmd/<module>_routes.go. Modules without CRUD expose New().
Console applications wire modules explicitly.

## Skip CRUD

```bash
modkit module create underwriting --no-crud
modkit entity create decision --module underwriting
```

The module policy persists: all its entities generate only domain models and
events, without repositories, services, handlers, or automatic migrations.
--create-all does not override --no-crud. Explicitly adding HTTP to a no-CRUD
module is rejected. entity create --no-crud skips CRUD for one entity in an
otherwise CRUD-enabled module.

Fields support string, bool, int, int64, uint, uint64, float64; ID is a
client-supplied string. Repeat --field name:type. Hyphenated names become
PascalCase Go fields and snake_case JSON fields.

entity create --force overwrites that entity's files only; it cannot change
existing CRUD mode. Existing modules cannot be recreated; extend them through
entity create and module add-element.

## HTTP API

CRUD entities expose POST/GET /<module>/<entity> and GET/PUT/DELETE
/<module>/<entity>/:id. POST takes JSON; PUT replaces all fields including zero
values. Status codes: 201 create, 200 read/update, 204 delete; 400 invalid input,
404 missing record, 409 duplicate.

```bash
curl 'localhost:8080/origination/application?limit=20&offset=0'
```

Lists return items, limit, offset, total. Limit defaults to 20 (1–100);
offset defaults to 0 (non-negative records to skip). Results order by ID.
COUNT and data queries are separate, so concurrent writes may affect totals.

## Optional elements

```bash
modkit module create documents --create-all
modkit module add-element origination http
modkit module add-element origination messaging
modkit module add-element origination outbox
modkit module add-element origination threading
modkit module add-element origination observability
modkit module list
modkit module validate origination
modkit entity create --help
modkit version
modkit system
```

Messaging, outbox, and observability provide extension contracts, not complete
integrations. Threading provides coordinated workers. HTTP adds handlers for
existing CRUD entities. Console consumers must resolve Gin dependencies and
wire a router explicitly when enabling HTTP.

## Compatibility and verification

This replaces the earlier entities/<name>/ and interfaces/http/ layout for new
scaffolds. Older modules without the new manifest are rejected, not migrated
automatically. Generate a reference module in a temporary project and transfer
custom rules, imports, composition, routes, and persistence mappings explicitly.
Do not add a new manifest to old-layout code.

Run go test ./... for generator checks. Set MODKIT_TEST_POSTGRES_DSN to run
TestPostgresCRUD against an isolated schema inside a rolled-back transaction.
The test covers CRUD, pagination, multiple shared-layer entities, and no-CRUD
route exclusions. See [the user guide](docs/user-guide.md) for configuration and integration details.


## Use cases, previews and optional infrastructure

```bash
modkit usecase create submit-application --module origination --entity application --dry-run
modkit usecase create submit-application --module origination --entity application
modkit platform list
modkit platform add audit --module origination
modkit platform add redis --module origination
modkit platform add sns
modkit platform add sqs --module origination
modkit platform add minio --module origination
go mod tidy
```

Use cases generate explicit Input/Output/Execute types. Execute returns a
not-implemented error until business rules are supplied; wire it in module.go.
--entity optionally injects an existing CRUD repository. All generation commands
accept --dry-run, which validates and lists file actions without writes or network.

PostgreSQL remains the default. Audit, Redis, SNS, SQS and S3/MinIO are opt-in.
--module adds explicit module integration; clients are not opened automatically.
Redis uses REDIS_URL. AWS adapters use AWS_REGION and the standard credentials
chain. SNS_ENDPOINT, SQS_ENDPOINT, S3_ENDPOINT support custom endpoints.
MinIO uses the S3 adapter with S3_PATH_STYLE=true.

Audit captures actor/tenant/request identity, entity ID, action and UTC timestamp
inside the business transaction. Failure rolls back the write. Identity comes
from trusted code via audit.WithIdentity; default actor is anonymous.
No request bodies are copied, and authentication/tenant isolation are not provided.
Existing custom repositories are not overwritten when enabling audit.

## Explicit migrations and merge drafts

New CRUD entities generate versioned SQL; AutoMigrate is removed.

```bash
modkit migration create add-application-status
modkit migration check
go run ./tools/migrate status
go run ./tools/migrate up
modkit migration merge reconcile --source 20261001120000_a.sql --source 20261001120000_b.sql
```

Migration drafts fail until reviewed and implemented. Goose applies migrations
explicitly with a PostgreSQL session lock; down reverts the last migration.
SQL is embedded: rebuild executables after migration changes.
Check detects structural conflicts, not SQL semantics or applied-history drift.
Merge prepares a non-executable draft under migrations/merge/, preserving source
files and recording their hashes. It does not automatically fix incompatible SQL
or renumber applied history. No database checksum verification is added.

Module manifests now use Version 2. Older AutoMigrate scaffolds need explicit
migration. --force cannot change CRUD field schemas; create a new migration and
update the model instead.

See [the user guide](docs/user-guide.md) for configuration, limitations and migration procedures.
Optional adapter tests use local protocol simulators, not live AWS/MinIO:

```bash
MODKIT_TEST_ADAPTERS=1 go test ./cmd -run TestAdapterIntegrations -v
```

Implementation references: [Goose provider](https://pressly.github.io/goose/documentation/provider/),
[Redis Go client](https://redis.io/docs/latest/develop/clients/go/connect/),
[AWS SDK for Go v2](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/welcome.html).

### Debug tracing (Jaeger)

New projects include `internal/platform/telemetry`, following the BFF and LOS_CORE
`Setup` / `Start` convention with OpenTelemetry and OTLP/HTTP. Export these variables
(the generated `.env.example` is documentation; it is not loaded automatically):

```sh
export DEBUG=true
export OTEL_SERVICE_NAME=loan-origination
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

`DEBUG` defaults to false. Disabled mode creates no exporter or recording spans;
HTTP middleware and each `Start` check the gate. Boolean values follow Go's
`strconv.ParseBool`; invalid values fail startup. The signal-specific
`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` takes precedence and must include the full
trace path. Otherwise `/v1/traces` is appended to the base endpoint, which defaults
to `http://localhost:4318`. Jaeger must expose its OTLP HTTP receiver.
Invalid endpoint configuration disables tracing with a diagnostic. Export failures
discard the failed batch without failing business operations; later batches retry
delivery. HTTP shutdown drains requests and flushes tracing with bounded timeouts.

Generated HTTP handlers, application services, use cases, repositories and optional
platform operations receive spans using the propagated context. HTTP middleware
extracts W3C trace context, excludes `/healthz`, and records route templates and
status codes. Span names are static: no SQL, credentials, payloads or entity IDs.
Pure domain objects and constructors remain independent of telemetry. Extend
business operations using `ctx, span := telemetry.Start(ctx, "static.operation")`
and `defer span.End()`. Pass `ctx` to downstream dependencies.

### Generated unit tests

Modules include domain event tests. Entities include domain tests and, when CRUD
is enabled, service, repository and HTTP tests (HTTP projects only). Use cases
include a cancellation test to extend with business scenarios. Services use fakes;
repositories use `go-sqlmock`; handlers use `httptest`. Telemetry tests validate
parent/child spans, the DEBUG gate, and OTLP export against a local test collector.
No external PostgreSQL or Jaeger instance is required.

```sh
go mod tidy
go test ./...
```

`--no-crud` generates only the applicable domain tests. Existing entity and use-case
test files are preserved, including with `--force`. This generation applies to new
artifacts; existing applications are not automatically rewritten. To validate the
CLI and both generated project variants, run:

```sh
MODKIT_TEST_GENERATED=1 go test ./...
```

## Generated documentation

`modkit init` includes `docs/code-documentation.md`. Each new module includes a
`README.md` to document business boundaries, rules, contracts, persistence,
integrations and tests. Entities, services, repositories, handlers and use cases
include comments and specific `TODO(doc)` markers. Replace them with actual
contracts while implementing each capability; they are not automatically provided
guarantees. The module README is created once and is preserved when entities or
use cases are added. All generated documentation is in English.
