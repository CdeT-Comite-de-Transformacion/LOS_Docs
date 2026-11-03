# modkit starter

CLI Go para monolitos modulares con DDD parcial, PostgreSQL, GORM y Gin.
Los modulos representan capacidades del negocio; las entidades comparten las
capas del modulo. Crear un modulo no crea una entidad del mismo nombre.

## Compilar

```bash
go build -o bin/modkit .
go test ./...
```

En Windows: `go build -o bin/modkit.exe .`.

## Crear un aplicativo

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

`init` pregunta el nombre si no se proporciona `--name`.
El template predeterminado es HTTP, escucha en :8080 y expone GET /healthz.
Usa `init --template console` para una aplicacion sin servidor HTTP.
La conexion PostgreSQL compartida esta en
`internal/platform/database/database.go`; DB_DSN es obligatorio.
No se carga un archivo .env automaticamente.

## Estructura del modulo

Tras crear las entidades del ejemplo:

```text
internal/
├── platform/database/database.go
└── origination/
    ├── .modkit.json
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

- domain: modelos, reglas, eventos y contratos.
- application: coordinacion de operaciones y publicacion de eventos.
- infrastructure: repositorios GORM e integraciones.
- http: rutas, validacion de entrada y respuestas.

Las carpetas se crean al necesitarlas. Las entidades no tienen subarboles propios.
Agregar otra entidad agrega archivos con su nombre dentro de estas mismas capas.

`module.go` queda disponible para composicion personalizada.
`module_generated.go` contiene Module, sus servicios, New y RegisterRoutes
(si corresponde). Se regenera al agregar entidades. No lo edites.
`.modkit.json` registra entidades, archivos y la politica CRUD; debe versionarse.
Agregar entidades conserva los archivos existentes y la composicion personalizada.

Para modulos con CRUD, `New(db)` devuelve `(*Module, error)`, construye sus servicios sin modificar el esquema de la BD. En HTTP, `cmd/<modulo>_routes.go` registra
el modulo una sola vez con la conexion compartida. En console, invoca New(db)
desde tu composicion. Un modulo sin entidades CRUD expone `New() *Module`.

## --no-crud

```bash
modkit module create underwriting --no-crud
modkit entity create decision --module underwriting
```

La politica se guarda para ese modulo. Sus entidades generan solo modelo y
eventos de dominio: sin repositorio, servicio, handler ni migracion automatica.
No afecta a otros modulos. `--create-all` no anula esta politica.
Agregar el elemento http explicitamente a un modulo --no-crud devuelve un error.

Tambien puedes omitir CRUD para una sola entidad:

```bash
modkit entity create applicant --module origination --no-crud
```

Esto permite implementar operaciones como AddApplicant o SubmitApplication con
reglas del dominio, en lugar de ofrecer CRUD independiente para cada entidad.
El generador no implementa esas reglas ni relaciones automaticamente.

## Entidades y campos

```bash
modkit entity create product --module catalog --field name:string --field active:bool
modkit entity create --help
```

Crea primero catalog con `modkit module create catalog`.
Tipos admitidos: string, bool, int, int64, uint, uint64, float64.
El ID string lo proporciona el cliente. Usa first-name para generar FirstName
en Go y first_name en JSON. Se rechazan nombres y campos duplicados o reservados.

`entity create --force` sobrescribe solo los archivos de esa entidad, preservando
los compartidos y otras entidades. No permite cambiar el modo CRUD de una
entidad existente: esa conversion requiere migracion explicita.
Un modulo existente no se recrea; amplialo con entity create o add-element.

## CRUD y paginacion

```bash
curl -X POST localhost:8080/origination/application -H 'Content-Type: application/json' -d '{"id":"a1","amount":1000}'
curl 'localhost:8080/origination/application?limit=20&offset=0'
curl localhost:8080/origination/application/a1
curl -X PUT localhost:8080/origination/application/a1 -H 'Content-Type: application/json' -d '{"amount":0}'
curl -X DELETE localhost:8080/origination/application/a1
```

POST crea, GET consulta/lista, PUT reemplaza campos (incluidos cero y false) y
DELETE elimina. Respuestas: 201, 200, 204; errores de entrada 400, inexistentes 404,
duplicados 409. Los eventos Created se publican con un NoopPublisher por defecto.

El listado devuelve items, limit, offset y total. Limit: 20 por defecto, de 1 a
100. Offset: 0 por defecto, no negativo, cantidad de registros a omitir.
Orden por ID; offset posterior al total devuelve items vacio. Se aplica COUNT,
LIMIT y OFFSET. Conteo y datos son consultas separadas; escrituras concurrentes
pueden cambiar el resultado entre ambas.

Los tipos de cada entidad tienen nombre propio: ApplicationService,
ApplicationRepository, ApplicationHandler y ApplicationPage. Pagination, los
errores y contratos de eventos se comparten dentro del modulo.

## Elementos opcionales y configuracion

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

messaging genera un publicador base y contrato de consumidor; outbox un contrato
de almacenamiento; threading un grupo de goroutines con cancelacion compartida;
observability contratos para instrumentacion. Son puntos de extension, no
integraciones completas. http agrega handlers a entidades CRUD existentes.
En proyectos console, los handlers requieren incorporar Gin con go mod tidy y
conectar el router desde tu composicion.

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

Validate comprueba los archivos registrados, no la conectividad ni reglas de negocio.
Los modulos no deben acceder a repositorios de otros modulos; utiliza contratos
de aplicacion o eventos. GORM implementa persistencia; las migraciones SQL versionadas administran el esquema;
las entidades CRUD incluyen etiquetas GORM sin importar el framework.
Puedes separar modelos persistidos al necesitarlo. AutoMigrate ya no se ejecuta.

## Pruebas y compatibilidad

```bash
go test ./...
MODKIT_TEST_POSTGRES_DSN="$DB_DSN" go test ./cmd -run TestPostgresCRUD -v
```

Las pruebas CRUD revierten su transaccion. Las pruebas del ejecutor de migraciones
usan otro schema aislado, que eliminan al terminar. Comprueban CRUD, paginacion,
auditoria, multiples entidades y ausencia de rutas para entidades --no-crud.

Este esquema reemplaza entities/<nombre>/ y interfaces/http/ en scaffolds nuevos.
No migra automaticamente proyectos anteriores: se rechazan modulos sin el
manifiesto de la estructura nueva para evitar mezclar ambas estructuras.
Para migrarlos, genera un modulo equivalente en un proyecto temporal, traslada
reglas y adaptadores y ajusta imports, composicion y rutas. Conserva los nombres
de tabla existentes al migrar datos. No copies el manifiesto sobre el codigo viejo.


## Casos de uso

```bash
modkit usecase create submit-application --module origination --entity application --dry-run
modkit usecase create submit-application --module origination --entity application
modkit usecase create record-decision --module underwriting
```

Se genera application/submit-application_usecase.go con SubmitApplicationInput,
SubmitApplicationOutput, SubmitApplicationUseCase y Execute. --entity es opcional:
inyecta el contrato de repositorio de una entidad CRUD existente.
Sin --entity puedes definir tus propias dependencias.

Execute devuelve un error explicito de implementacion pendiente hasta que agregues
las reglas. Conecta el caso de uso desde module.go; no se expone una ruta
automaticamente. --force reemplaza solo el archivo del caso de uso.

## Vista previa

Todos los comandos que generan archivos aceptan --dry-run: init, module create,
module add-element, entity create, usecase create, platform add, migration create
y migration merge. La salida lista CREATE, UPDATE o UNCHANGED. La validacion es
la misma, pero no escribe archivos/directorios, no descarga SDK ni conecta servicios.
La vista previa de init usa el nombre por defecto si no se proporciona --name.

## Migraciones SQL y conflictos

Los proyectos nuevos incluyen Goose y migrations/sql/, embebido en el ejecutable.
Crear una entidad CRUD genera una migracion CREATE TABLE con Up/Down; --no-crud
no genera tablas. New(db) ya no llama AutoMigrate.

```bash
modkit migration create add-application-status
# Implementa y revisa Up/Down en el archivo creado.
modkit migration check
go run ./tools/migrate status
go run ./tools/migrate up
# Revertir la ultima migracion, solo cuando corresponda:
go run ./tools/migrate down
```

El ejecutor usa un advisory lock PostgreSQL para serializar ejecuciones. Goose
aplica cada migracion transaccionalmente por defecto; no incluyas NO TRANSACTION
sin evaluar su comportamiento. Ejecuta migraciones explicitamente antes de
iniciar la aplicacion. Al estar embebidas, recompila despues de agregar SQL.

migration create deja una barrera SQL que falla hasta completar la migracion,
evitando registrar como aplicada una migracion vacia. migration check detecta
nombres invalidos, versiones duplicadas, secciones faltantes, marcadores de
conflicto Git y migraciones pendientes de revision. Es una validacion estructural,
no valida semanticamente SQL ni comprueba historial aplicado.

```bash
modkit migration merge reconcile-application   --source 20261001120000_branch-a.sql   --source 20261001120000_branch-b.sql --dry-run
```

merge prepara migrations/merge/reconcile-application.sql: concatena Up en el
orden indicado y Down en orden inverso, e incluye hashes de los originales.
Ese directorio no se ejecuta ni se embebe. No modifica ni elimina fuentes.

El borrador requiere resolver SQL incompatible. Antes de reemplazar o renumerar
migraciones pendientes, verifica que no se aplicaron en ningun entorno compartido.
Si ya se aplicaron, crea una nueva migracion correctiva. No hay un fixer que
reescriba automaticamente SQL o historial aplicado. Los marcadores Git deben
resolverse antes de combinar fuentes.

--force en entidades permite regenerar codigo con el mismo esquema, pero rechaza
cambios de campos CRUD: usa una migracion nueva y actualiza el modelo explicitamente.
Goose lleva el historial de versiones; no se agrega verificacion de checksum
contra la BD para detectar ediciones de migraciones ya aplicadas.

## Componentes transversales opcionales

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

PostgreSQL sigue siendo el default. Los demas componentes son opt-in; --create-all
de un modulo no los instala. Sin --module se agrega el adaptador compartido al
proyecto; con --module se agrega tambien su fabrica explicita en infrastructure/.
Instancia clientes una vez desde module.go y reutilizalos. No se crean recursos
remotos, buckets, topics, suscripciones o colas. Repetir platform add conserva
adaptadores existentes y no vuelve a escribir personalizaciones.

| Componente | Ubicacion | Configuracion |
|---|---|---|
| PostgreSQL | platform/database | DB_DSN |
| Auditoria | platform/audit | Identidad en context; tabla mediante migracion |
| Redis/cache | platform/cache | REDIS_URL; redis:// o rediss:// |
| SNS/topics | platform/messaging | AWS_REGION, cadena de credenciales AWS; SNS_ENDPOINT opcional |
| SQS/colas | platform/messaging | AWS_REGION, cadena de credenciales AWS; SQS_ENDPOINT opcional |
| S3/MinIO | platform/storage | AWS_REGION; S3_ENDPOINT y S3_PATH_STYLE para MinIO |

MinIO usa el mismo adaptador S3. Configura S3_ENDPOINT, S3_PATH_STYLE=true,
AWS_REGION y credenciales AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY compatibles
con tu servidor. En AWS, el SDK admite roles IAM y la cadena normal de credenciales.

Redis implementa Get/Set/Delete, TTL, ErrMiss y Close. TTL cero conserva la clave
sin expiracion; invalida cache despues de confirmar cambios en BD.

SNS publica un envelope JSON con ID, Body (base64), Attributes, GroupID y
DeduplicationID. SQS implementa Send, Receive, Ack y ExtendVisibility usando el
mismo envelope. Attributes son datos del envelope, no atributos nativos para
filtros SNS. Para SNS -> SQS configura RawMessageDelivery en la suscripcion.
Ack usa el receipt handle, no el ID del mensaje. Los consumidores deben ser
idempotentes; reintentos, DLQ, suscripciones y politicas de cola se configuran en
infraestructura. Si Receive devuelve resultados parciales y error de decodificacion,
procesa los resultados validos y deja el mensaje defectuoso para reintento/DLQ.

S3/MinIO implementa Put/Get/Delete y PresignGet. El llamador cierra el Body de Get.
Put es una carga simple; para cargas grandes agrega multipart upload.
Las URL firmadas expiran en hasta siete dias (las credenciales temporales pueden
vencer antes). Los endpoints personalizados permiten simuladores locales.

## Auditoria transaccional

`platform add audit --module origination` genera la tabla mediante migracion y
activa auditoria create/update/delete en repositorios CRUD del modulo, incluidos
los que se agreguen despues. Si un repositorio existente fue personalizado, el
comando se detiene sin sobrescribirlo: integra audit.Record manualmente en su
transaccion. A nivel proyecto instala la libreria para usarla explicitamente.

Cada registro guarda fecha UTC, actor, tenant, request ID, modulo, entidad,
ID y accion. No copia cuerpos de peticion, documentos ni valores sensibles.
Si la insercion de auditoria falla, revierte la escritura de negocio.
No registra lecturas, rechazos ni operaciones externas automaticamente.

Propaga identidad desde autenticacion validada o desde el worker:

```go
ctx = audit.WithIdentity(ctx, audit.Identity{
    ActorID: authenticatedUserID,
    TenantID: tenantID,
    RequestID: requestID,
})
```

Sin identidad, el actor queda como anonymous. No se confia en headers de actor.
Esto no implementa autenticacion, autorizacion ni aislamiento multi-tenant.
La API de auditoria solo inserta; permisos DB, retencion e inmutabilidad fuerte
deben configurarse segun el despliegue.

Para otras operaciones usa audit.Record dentro de la transaccion del caso de uso.
Publicar a SNS/SQS no forma parte de una transaccion PostgreSQL: un outbox
transaccional con dispatcher sigue siendo una extension por implementar.

## Verificacion ampliada

```bash
go test ./...
MODKIT_TEST_POSTGRES_DSN="$DB_DSN" go test ./cmd -run 'TestPostgresCRUD|TestGeneratedConsoleBuild' -v
MODKIT_TEST_ADAPTERS=1 go test ./cmd -run TestAdapterIntegrations -v
```

Los adaptadores se prueban contra Redis simulado y endpoints HTTP locales,
sin cuentas AWS ni un servidor MinIO real. Incluye TTL, cache miss, envelope de
mensajes, receipt handles, operaciones de objetos y firma de URL.

El manifiesto de modulo usa ahora Version 2 por las migraciones SQL explicitas.
Los modulos anteriores requieren migracion manual; genera un proyecto temporal
de referencia, conserva reglas y datos y adapta la composicion. No cambies solo
el numero del manifiesto.

## Tracing y pruebas por defecto

Las plantillas siguen el patrón `telemetry.Setup` / `telemetry.Start` del BFF y
LOS_CORE, con OpenTelemetry por OTLP/HTTP hacia Jaeger:

```bash
export DEBUG=true
export OTEL_SERVICE_NAME=loan-origination
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

Sin `DEBUG` o con `DEBUG=false`, no se crea exportador ni se graban spans.
`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` tiene prioridad sobre el endpoint general.
El archivo `.env.example` generado documenta las variables; no se carga solo.
Los fallos del colector no interrumpen las operaciones del aplicativo.
HTTP propaga el contexto W3C y el cierre del servidor vacía el exportador.
Los spans usan nombres estáticos, sin datos de negocio, credenciales ni SQL.

Cada módulo incluye pruebas de eventos. Cada entidad genera pruebas de dominio
y, si tiene CRUD, de servicio, repositorio y handler HTTP. Los casos de uso
incluyen una prueba de cancelación como punto de partida. Los tests usan fakes,
SQL simulado y servidores locales temporales; no necesitan BD ni Jaeger externos.
Los tests existentes se conservan incluso usando `--force`.

```bash
go mod tidy
go test ./...
```

Los proyectos ya creados no se reescriben automáticamente. Para verificar el CLI,
las plantillas HTTP/consola y los adaptadores opcionales:

```bash
MODKIT_TEST_GENERATED=1 MODKIT_TEST_ADAPTERS=1 go test ./...
```

## Documentación generada

`modkit init` incluye `docs/code-documentation.md`. Cada nuevo módulo incluye un
`README.md` para documentar límites de negocio, reglas, contratos, persistencia,
integraciones y pruebas. Entidades, servicios, repositorios, handlers y casos de
uso incluyen comentarios y marcadores `TODO(doc)` específicos. Sustituirlos por
los contratos reales al implementar cada capacidad; no representan garantías
automáticamente implementadas. El README del módulo se crea una sola vez y no
se reescribe al añadir entidades o casos de uso.
