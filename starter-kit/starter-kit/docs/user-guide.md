# modkit user guide

A Go CLI for modular monoliths using selected DDD patterns, PostgreSQL, GORM and
Gin. Modules represent business capabilities; entities share the module's layers.
Creating a module does not create an entity with the same name.

## Build

```bash
go build -o bin/modkit .
go test ./...
```

On Windows, use `go build -o bin/modkit.exe .`.

## Create an application

```bash
modkit init --name loan-origination
modkit module create origination
modkit entity create application --module origination --field amount:int64
modkit entity create applicant --module origination --field name:string --no-crud
go mod tidy
export DB_DSN='host=localhost port=5433 user=postgres password=example dbname=postgres sslmode=disable'
go run ./tools/migrate up
go run .
```

`init` prompts for a name when `--name` is omitted. The default HTTP template
listens on :8080 and exposes GET /healthz. Use `init --template console` for an
application without an HTTP server. The shared PostgreSQL connection lives in
`internal/platform/database/database.go`; `DB_DSN` is required. Environment files
are not loaded automatically.

## Module structure

After creating the example entities:

```text
internal/
├── platform/database/database.go
└── origination/
    ├── .modkit.json
    ├── README.md
    ├── module.go
    ├── module_generated.go
    ├── domain/
    │   ├── application.go
    │   ├── application_events.go
    │   ├── application_repository.go
    │   ├── applicant.go
    │   ├── applicant_events.go
    │   ├── errors.go
    │   ├── events.go
    │   └── pagination.go
    ├── application/
    │   ├── application_service.go
    │   └── ports.go
    ├── infrastructure/
    │   └── application_repository.go
    └── http/
        └── application_handler.go
```

The tree omits generated `*_test.go` files for brevity.

- `domain`: models, rules, events and contracts.
- `application`: operation coordination and event publication.
- `infrastructure`: GORM repositories and integrations.
- `http`: routes, input validation and responses.

Directories are created as needed. Entities do not receive independent directory
trees; another entity adds named files to the same layers.

`module.go` is available for custom composition. `module_generated.go` contains
Module, services, New and RegisterRoutes when applicable. It is regenerated when
entities are added; do not edit it. Commit `.modkit.json`, which tracks entities,
files and CRUD policy. Adding entities preserves existing files and custom wiring.

For CRUD modules, `New(db)` returns `(*Module, error)` and constructs services
without modifying the database schema. In HTTP projects, `cmd/<module>_routes.go`
registers the module once with the shared connection. In console projects, call
New(db) from your composition root. Modules without CRUD entities expose
`New() *Module`.

## Disable CRUD

```bash
modkit module create underwriting --no-crud
modkit entity create decision --module underwriting
```

The policy is stored for that module. Its entities generate only domain models,
events and applicable tests: no repository, service, handler or automatic migration.
Other modules are unaffected. `--create-all` does not override this policy.
Explicitly adding HTTP to a `--no-crud` module returns an error.

To omit CRUD for one entity only:

```bash
modkit entity create applicant --module origination --no-crud
```

This supports domain operations such as AddApplicant or SubmitApplication instead
of independent CRUD for every entity. The generator does not implement those
business rules or relationships automatically.

## Entities and fields

```bash
modkit module create catalog
modkit entity create product --module catalog --field name:string --field active:bool
modkit entity create --help
```

Supported types: string, bool, int, int64, uint, uint64 and float64. The client
supplies the string ID. Use `first-name` to generate `FirstName` in Go and
`first_name` in JSON. Duplicate or reserved names and fields are rejected.

`entity create --force` overwrites that entity's generated source files while
preserving shared files, other entities and existing tests. Changing an existing
entity's CRUD mode requires an explicit migration. Existing modules cannot be
recreated; extend them with `entity create` or `add-element`.

## CRUD and pagination

```bash
curl -X POST localhost:8080/origination/application -H 'Content-Type: application/json' -d '{"id":"a1","amount":1000}'
curl 'localhost:8080/origination/application?limit=20&offset=0'
curl localhost:8080/origination/application/a1
curl -X PUT localhost:8080/origination/application/a1 -H 'Content-Type: application/json' -d '{"amount":0}'
curl -X DELETE localhost:8080/origination/application/a1
```

POST creates, GET reads/lists, PUT replaces fields including zero and false, and
DELETE removes. Success codes are 201, 200 and 204. Invalid input returns 400,
missing records 404 and duplicates 409. Created events use NoopPublisher by default.

Lists return items, limit, offset and total. Limit defaults to 20 and accepts
1–100. Offset defaults to 0 and is the non-negative number of records to skip.
Results are ordered by ID; offsets beyond the total yield an empty items list.
COUNT, LIMIT and OFFSET are applied. Counting and reading are separate queries;
concurrent writes can change the result between them.

Entity types have distinct names: ApplicationService, ApplicationRepository,
ApplicationHandler and ApplicationPage. Pagination, errors and event contracts
are shared within the module.

## Optional elements and configuration

```bash
modkit module create documents --create-all
modkit module add-element origination messaging
modkit module add-element origination outbox
modkit module add-element origination threading
modkit module add-element origination observability
modkit module add-element origination http
modkit module list
modkit module validate origination
modkit version
modkit system
```

`messaging` generates a base publisher and consumer contract; `outbox` a storage
contract; `threading` a goroutine group with shared cancellation; `observability`
instrumentation contracts. These are extension points, not complete integrations.
HTTP adds handlers to existing CRUD entities. Console projects must include Gin
with `go mod tidy` and wire their router explicitly.

```yaml
project:
  name: loan-origination
  module: github.com/your-org/loan-origination
  architecture: modular-ddd
  template: http
modules:
  root: internal
defaults:
  template: console
```

Validation checks registered files, not connectivity or business rules. Modules
must use application contracts or events instead of accessing another module's
repositories. GORM handles persistence; versioned SQL migrations manage schema.
CRUD entities include GORM tags without importing the framework. Separate
persistence models when needed. AutoMigrate is no longer executed.

## Tests and compatibility

```bash
go test ./...
MODKIT_TEST_POSTGRES_DSN="$DB_DSN" go test ./cmd -run TestPostgresCRUD -v
```

CRUD tests roll back their transaction. Migration runner tests use another
isolated schema and remove it afterward. They check CRUD, pagination, auditing,
multiple entities and the absence of routes for `--no-crud` entities.

New scaffolds replace `entities/<name>/` and `interfaces/http/`. Earlier projects
are not migrated automatically: modules without the new structure's manifest are
rejected to avoid mixing layouts. Generate an equivalent module in a temporary
project, transfer rules and adapters, and adjust imports, composition and routes.
Preserve existing table names when migrating data. Do not copy the new manifest
over old code.

## Use cases

```bash
modkit usecase create submit-application --module origination --entity application --dry-run
modkit usecase create submit-application --module origination --entity application
modkit usecase create record-decision --module underwriting
```

Generates `application/submit-application_usecase.go` with SubmitApplicationInput,
SubmitApplicationOutput, SubmitApplicationUseCase and Execute. Optional `--entity`
injects an existing CRUD entity's repository contract. Without it, define your
own dependencies.

Execute returns an explicit not-implemented error until business rules are added.
Wire the use case in `module.go`; no route is exposed automatically. `--force`
replaces the use-case source file while preserving its existing test file.

## Preview changes

All file-generating commands accept `--dry-run`: init, module create, module
add-element, entity create, usecase create, platform add, migration create and
migration merge. Output lists CREATE, UPDATE or UNCHANGED. Validation is identical,
but no files/directories are written, SDKs downloaded or services contacted.
The init preview uses the default project name when `--name` is omitted.

## SQL migrations and conflicts

New projects include Goose and `migrations/sql/`, embedded in the executable.
Creating a CRUD entity generates a CREATE TABLE migration with Up/Down sections;
`--no-crud` generates no table. New(db) no longer calls AutoMigrate.

```bash
modkit migration create add-application-status
# Implement and review Up/Down in the generated file.
modkit migration check
go run ./tools/migrate status
go run ./tools/migrate up
# Roll back the latest migration only when appropriate:
go run ./tools/migrate down
```

The runner uses a PostgreSQL advisory lock to serialize executions. Goose applies
migrations transactionally by default; evaluate behavior before using NO TRANSACTION.
Run migrations explicitly before application startup. Rebuild after adding SQL
because migrations are embedded.

`migration create` inserts an SQL guard that fails until the migration is completed,
preventing an empty migration from being recorded as applied. `migration check`
detects invalid names, duplicate versions, missing sections, Git conflict markers
and migrations awaiting review. It is structural validation, not SQL semantic
validation or verification of applied history.

```bash
modkit migration merge reconcile-application --source 20261001120000_branch-a.sql --source 20261001120000_branch-b.sql --dry-run
```

Merge prepares `migrations/merge/reconcile-application.sql`: Up sections follow
source order and Down sections use reverse order; original hashes are included.
This directory is neither executed nor embedded. Sources are not changed or deleted.

Resolve incompatible SQL in the draft. Before replacing or renumbering pending
migrations, verify that none have been applied in any shared environment. If already
applied, create a corrective migration. No fixer automatically rewrites SQL or
applied history. Resolve Git conflict markers before merging sources.

Entity `--force` permits regeneration with the same schema but rejects CRUD field
changes: add a migration and update the model explicitly. Goose tracks versions;
no database checksum verification is added to detect edits to applied migrations.

## Optional shared components

```bash
modkit platform list
modkit platform add audit --module origination
modkit platform add redis --module origination
modkit platform add sns
modkit platform add sqs --module underwriting
modkit platform add minio --module documents
modkit platform list --module origination
go mod tidy
go run ./tools/migrate up
```

PostgreSQL remains the default. Other components are opt-in; module `--create-all`
does not install them. Without `--module`, the shared project adapter is added;
with it, an explicit factory is also added in the module's infrastructure layer.
Create clients once in `module.go` and reuse them. No remote resources, buckets,
topics, subscriptions or queues are created. Repeating `platform add` preserves
existing adapters and customizations.

| Component | Location | Configuration |
| --- | --- | --- |
| PostgreSQL | platform/database | DB_DSN |
| Audit | platform/audit | Context identity; table created through migration |
| Redis/cache | platform/cache | REDIS_URL; redis:// or rediss:// |
| SNS/topics | platform/messaging | AWS_REGION, AWS credential chain; optional SNS_ENDPOINT |
| SQS/queues | platform/messaging | AWS_REGION, AWS credential chain; optional SQS_ENDPOINT |
| S3/MinIO | platform/storage | AWS_REGION; S3_ENDPOINT and S3_PATH_STYLE for MinIO |

MinIO uses the S3 adapter. Set S3_ENDPOINT, S3_PATH_STYLE=true, AWS_REGION and
AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY compatible with your server. On AWS, the
SDK supports IAM roles and the standard credential chain.

Redis implements Get/Set/Delete, TTL, ErrMiss and Close. Zero TTL keeps the key
without expiration. Invalidate cached values after database changes commit.

SNS publishes a JSON envelope with ID, Body (base64), Attributes, GroupID and
DeduplicationID. SQS implements Send, Receive, Ack and ExtendVisibility with the
same envelope. Attributes are envelope data, not native SNS filtering attributes.
Configure RawMessageDelivery on SNS-to-SQS subscriptions. Ack uses the receipt
handle, not the message ID. Consumers must be idempotent; retries, DLQs,
subscriptions and queue policies are configured in infrastructure. If Receive
returns partial results and a decoding error, process valid results and leave
the malformed message for retry/DLQ handling.

S3/MinIO implements Put/Get/Delete and PresignGet. Callers close the Body returned
by Get. Put is a simple upload; add multipart support for large uploads. Signed
URLs expire within seven days; temporary credentials may expire earlier. Custom
endpoints support local simulators.

## Transactional auditing

`platform add audit --module origination` generates a table migration and enables
create/update/delete auditing in the module's CRUD repositories, including future
entities. If an existing repository was customized, the command stops without
overwriting it: integrate audit.Record manually in its transaction. Project-level
installation provides the library for explicit use.

Records contain UTC time, actor, tenant, request ID, module, entity, ID and action.
Request bodies, documents and sensitive values are not copied. Audit insertion
failure rolls back the business write. Reads, rejections and external operations
are not audited automatically.

Propagate identity from validated authentication or a worker:

```go
ctx = audit.WithIdentity(ctx, audit.Identity{
    ActorID: authenticatedUserID,
    TenantID: tenantID,
    RequestID: requestID,
})
```

Without identity, the actor is anonymous. Actor headers are not trusted. This does
not implement authentication, authorization or tenant isolation. The audit API
only inserts records; configure database permissions, retention and stronger
immutability guarantees for your deployment.

For other operations, call audit.Record within the use-case transaction. SNS/SQS
publication is outside a PostgreSQL transaction; a transactional outbox with a
dispatcher remains an extension to implement.

## Extended verification

```bash
go test ./...
MODKIT_TEST_POSTGRES_DSN="$DB_DSN" go test ./cmd -run 'TestPostgresCRUD|TestGeneratedConsoleBuild' -v
MODKIT_TEST_ADAPTERS=1 go test ./cmd -run TestAdapterIntegrations -v
```

Adapters are tested against simulated Redis and local HTTP endpoints, without AWS
accounts or a real MinIO server. Coverage includes TTL, cache misses, message
envelopes, receipt handles, object operations and URL signing.

The module manifest uses Version 2 for explicit SQL migrations. Earlier modules
require manual migration: generate a temporary reference project, retain business
rules and data, and adapt composition. Do not change only the manifest version.

## Default tracing and tests

Templates follow the BFF and LOS_CORE `telemetry.Setup` / `telemetry.Start` pattern,
using OpenTelemetry over OTLP/HTTP toward Jaeger:

```bash
export DEBUG=true
export OTEL_SERVICE_NAME=loan-origination
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

Without DEBUG or with DEBUG=false, no exporter is created and no spans are recorded.
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT overrides the general endpoint. The generated
`.env.example` documents variables but is not loaded automatically. Collector
failures do not interrupt application operations. HTTP propagates W3C context;
server shutdown flushes the exporter. Span names are static, without business
data, credentials or SQL.

Modules include event tests. Entities include domain tests and, with CRUD enabled,
service, repository and HTTP handler tests where applicable. Use cases include a
cancellation test as a starting point. Tests use fakes, simulated SQL and temporary
local servers; no external database or Jaeger instance is required. Existing tests
are preserved even with `--force`.

```bash
go mod tidy
go test ./...
```

Existing projects are not rewritten automatically. To verify the CLI, HTTP/console
templates and optional adapters:

```bash
MODKIT_TEST_GENERATED=1 MODKIT_TEST_ADAPTERS=1 go test ./...
```

## Generated documentation

`modkit init` includes `docs/code-documentation.md`. New modules include a README
for business boundaries, rules, contracts, persistence, integrations and tests.
Entities, services, repositories, handlers and use cases include comments and
specific `TODO(doc)` markers. Replace them with actual contracts when implementing
each capability; they are not automatically provided guarantees. Module READMEs
are created once and preserved when entities or use cases are added. Write all
content in English and use English kebab-case documentation filenames.
