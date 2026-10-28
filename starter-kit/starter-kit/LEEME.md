# modkit starter

Starter CLI en Go para aplicaciones DDD modulares y event-driven.

## Comandos incluidos

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

## Compilacion

```bash
go mod tidy
go build -o bin/modkit .
```

En Windows:

```powershell
go build -o bin/modkit.exe .
.\bin\modkit.exe system
```

## Inicializar un proyecto

Desde otro proyecto Go, ejecuta:

```bash
/path/to/modkit init
```

`init` solicita el nombre del proyecto y lo usa para `project.name` y el
modulo Go. Presiona Enter para usar el nombre predeterminado.

Para automatizaciones, proporciona el nombre directamente:

```bash
/path/to/modkit init --name payments-api
```

Esto crea el modulo `github.com/your-org/payments-api`.

### Templates de proyecto

El template de proyecto predeterminado es `http`:

```bash
/path/to/modkit init
go run .
```

Genera un servidor HTTP con Gin en `:8080` con `GET /healthz`. El proyecto
generado declara `github.com/gin-gonic/gin` en `go.mod`; ejecuta `go mod tidy`
despues de crear el proyecto para resolver sus dependencias.

Usa el template `console` cuando no necesites un servidor HTTP:

```bash
/path/to/modkit init --template console
```

Los templates disponibles se muestran con:

```bash
/path/to/modkit init --help
```

El template seleccionado se guarda en `project.template`. La propiedad
`defaults.template` controla de forma independiente el template de modulos.

## Configuracion

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

## Crear un modulo

```bash
/path/to/modkit module create credit-request
```

Por defecto, crea el nucleo del modulo y los contratos de eventos de dominio.
Usa `--create-all` para generar todos los elementos disponibles.

```bash
/path/to/modkit module create credit-request --create-all
```

Estructura predeterminada:

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

En un proyecto HTTP, la creacion del modulo tambien registra el handler en el
router raiz y genera `cmd/creditrequest_routes.go`. Las rutas son `POST /credit-request`, `GET /credit-request` y
`GET|PUT|DELETE /credit-request/:id`.

## Agregar elementos incrementalmente

```bash
/path/to/modkit module add-element credit-request http
/path/to/modkit module add-element credit-request messaging
/path/to/modkit module add-element credit-request outbox
/path/to/modkit module add-element credit-request threading
/path/to/modkit module add-element credit-request observability
```

Usa `modkit module add-element --help` para ver los elementos disponibles.

Con `--create-all`, los archivos opcionales incluyen:

```text
infrastructure/event_publisher.go
infrastructure/outbox.go
infrastructure/worker_group.go
interfaces/consumer/handler.go
interfaces/observability/hooks.go
```

## Guia de elementos del modulo

### Dominio

El dominio contiene entidades, eventos de dominio, contratos de repositorio y
errores de dominio. No debe importar bases de datos, brokers, HTTP ni frameworks.

Las entidades registran eventos con `Record` y los exponen con `PullEvents`:

```go
entity := domain.NewCreditRequest("request-1")
entity.Record(domain.NewCreditRequestCreated(entity.ID))
events := entity.PullEvents()
```

`event.go` define el nombre del evento, el ID del agregado y la fecha UTC.
Agrega un evento cuando una transicion de negocio sea relevante fuera del agregado.

### Aplicacion

`service.go` coordina el caso de uso: crea la entidad, registra su evento, la
guarda mediante el repositorio y publica los eventos pendientes.

```go
service := application.NewServiceWithPublisher(repository, publisher)
err := service.Create(ctx, "request-1")
```

`ports.go` define el puerto `EventPublisher` y un `NoopPublisher` para proyectos
que aun no han seleccionado un broker.

### Infraestructura

`repository.go` implementa CRUD con GORM y la conexion compartida a PostgreSQL.

`event_publisher.go` conecta el puerto de publicacion con Kafka, NATS, RabbitMQ,
SNS/SQS u otro broker.

`outbox.go` define `Append`, `Pending` y `MarkPublished`. En produccion, el
outbox debe persistir el evento en la misma transaccion que el agregado y
despacharlo de forma asincrona con reintentos e idempotencia.

### Concurrencia

`worker_group.go` coordina goroutines independientes bajo un mismo contexto.
El primer error cancela el contexto compartido; el grupo espera a todos los
workers y devuelve ese error.

```go
workers := infrastructure.NewWorkerGroup(
    func(ctx context.Context) error {
        for {
            select {
            case <-ctx.Done():
                return ctx.Err()
            default:
                // procesar una unidad de trabajo
            }
        }
    },
)
err := workers.Run(ctx)
```

El helper no proporciona cola, scheduler, politica de reintentos ni
sincronizacion de estado compartido. Usa channels o la primitiva `sync` adecuada.

### Interfaces

`interfaces/http/handler.go` define el limite HTTP de Gin y `RegisterRoutes`. Mapea
peticiones a casos de uso y resultados a respuestas HTTP.

`interfaces/consumer/handler.go` define el limite para eventos entrantes del
broker. Deserializa el mensaje y reconocelo solo despues de procesarlo correctamente.

`interfaces/observability/hooks.go` expone hooks para inicio/fin de comandos y
publicacion de eventos. Conectalos a logs, metricas, trazas y correlation IDs.

### Composicion

`module.go` conecta el repositorio y el servicio de aplicacion sin imponer una
base de datos o broker especifico al dominio.

## Validar un modulo

```bash
/path/to/modkit module validate credit-request
```

La validacion comprueba que existan los archivos base generados. No comprueba
conectividad runtime, migraciones, configuracion del broker ni invariantes de negocio.

## Extensiones recomendadas

- adaptadores de persistencia y mensajeria: PostgreSQL, Redis, Kafka, NATS, RabbitMQ, SNS/SQS
- outbox transaccional con dispatcher
- reintentos, idempotencia y colas dead-letter
- versionado de esquemas y contract tests
- observabilidad con logs, metricas, tracing y correlation IDs
- templates externos, presets, releases e imagenes de contenedor


## PostgreSQL y entidades CRUD

Los proyectos nuevos incluyen GORM y el driver PostgreSQL por defecto, tanto
HTTP como console. Configura una base existente mediante la variable de entorno
`DB_DSN`; `cmd/database.go` abre la conexion. No se carga un archivo .env automaticamente.

```bash
modkit init --name ventas
modkit module create sales
modkit entity create customer --module sales --field name:string --field active:bool
go mod tidy
export DB_DSN='host=localhost port=5433 user=postgres password=example dbname=postgres sslmode=disable'
go run .
```

La entidad principal del modulo trae CRUD. Las entidades adicionales se crean en
`internal/sales/entities/customer/`, con sus propias capas domain, application,
infrastructure e interfaces/http. Puedes repetir el comando para otras entidades.
`--field` acepta string, bool, int, int64, uint, uint64 y float64.
El ID string ya viene incluido y lo proporciona el cliente. Los nombres de campos
se convierten a PascalCase en Go y snake_case en JSON; usa nombres como first-name.
Los archivos existentes se protegen salvo que uses `--force`.

```bash
curl -X POST localhost:8080/sales/customer -H 'Content-Type: application/json' -d '{"id":"c1","name":"Ana","active":true}'
curl localhost:8080/sales/customer
curl localhost:8080/sales/customer/c1
curl -X PUT localhost:8080/sales/customer/c1 -H 'Content-Type: application/json' -d '{"name":"Ana","active":false}'
curl -X DELETE localhost:8080/sales/customer/c1
```

PUT reemplaza todos los campos, incluidos valores cero. Las respuestas son
201 al crear, 200 al consultar/actualizar y 204 al eliminar; los errores de entrada,
no encontrado y duplicado devuelven 400, 404 y 409. El listado incluye paginacion con limit y offset.

En HTTP, los modulos se inicializan con la conexion compartida y ejecutan
AutoMigrate antes de servir las rutas. En console, conecta cada modulo desde
tu caso de uso con `module.New(db)`, que devuelve `(*Module, error)`.
Las tablas se separan por modulo y entidad. AutoMigrate es el punto de partida;
para despliegues controlados puedes sustituirlo por migraciones versionadas.

Estos cambios aplican a scaffolds nuevos. Proyectos anteriores necesitan adaptar
`cmd/root.go`, agregar `cmd/database.go` y las dependencias antes de generar nuevas
rutas. POST ahora recibe JSON; `service.Create(ctx, id)` sigue disponible, y
`service.CreateEntity(ctx, entity)` permite enviar los campos adicionales.

Para ejecutar la prueba CRUD real sobre PostgreSQL (schema aislado dentro de
una transaccion que se revierte), desde starter-kit:

```bash
MODKIT_TEST_POSTGRES_DSN="$DB_DSN" go test ./cmd -run TestPostgresCRUD -v
```


## List pagination / Paginacion de listados

```bash
curl 'localhost:8080/sales/customer?limit=10&offset=10'
```

```json
{"items":[],"limit":10,"offset":10,"total":0}
```

Todos los listados CRUD generados (modulos y entidades) usan limit (20 por defecto,
entre 1 y 100) y offset (0 por defecto, no negativo). Offset indica cuantos
registros omitir y no necesita ser multiplo de limit. Valores vacios, no enteros
o fuera de rango devuelven 400. Un offset mas alla del total devuelve items vacio
conservando el total. El orden es por ID.
El repositorio aplica COUNT, LIMIT y OFFSET en PostgreSQL; las consultas de
conteo y datos son independientes, por lo que escrituras concurrentes pueden
cambiar los resultados entre consultas.

La firma del servicio es `List(ctx, domain.Pagination{Limit: 20, Offset: 0})`.
La respuesta HTTP pasa de un array a un objeto con items y metadatos.
Aplica a codigo generado nuevo; adapta consumidores y modulos existentes antes
de regenerarlos con --force, que sobrescribe sus archivos.
