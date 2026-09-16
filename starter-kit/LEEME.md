# modkit starter

Starter CLI en Go para aplicaciones DDD modulares y event-driven.

## Comandos incluidos

```bash
modkit init
modkit module create <name>
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
router raiz y genera `cmd/creditrequest_routes.go`. La ruta es
`POST /credit-request`.

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

`repository.go` es el punto de implementacion del adaptador de persistencia.
Conectalo a PostgreSQL, Redis u otro sistema de almacenamiento.

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
