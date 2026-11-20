---
sidebar_position: 5
title: Observabilidad
hide_title: true
---

<h1 align="center">Observabilidad</h1>

<div align="center">
  <img src="https://img.shields.io/badge/OpenTelemetry-425CC7?style=for-the-badge&logo=opentelemetry&logoColor=white" alt="OpenTelemetry"/>
  <img src="https://img.shields.io/badge/Jaeger-60D0E4?style=for-the-badge&logo=jaeger&logoColor=black" alt="Jaeger"/>
  <img src="https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=prometheus&logoColor=white" alt="Prometheus"/>
  <img src="https://img.shields.io/badge/Grafana_%C2%B7_Loki-F46800?style=for-the-badge&logo=grafana&logoColor=white" alt="Grafana · Loki"/>
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go"/>
</div>

<p align="center">
  <i>Trazas, métricas y logs correlacionados por traceId con OpenTelemetry y un Collector.</i>
</p>
<p align="center">
  <i><b>Estado: propuesta para revisión</b></i>
</p>
<br />

---

Propuesta para que LOS sea observable con OpenTelemetry (OTel): que a partir de un síntoma, como el código de soporte de un error, el equipo llegue a la causa usando solo trazas, logs y métricas, sin desplegar código para investigar.

## Punto de partida

El BFF y el Core ya tienen instrumentación de trazas: middleware HTTP en el BFF, interceptores gRPC con propagación W3C entre BFF y Core, y spans de negocio, Postgres, Redis y Keycloak. Para que esa base permita pasar del síntoma a la causa faltan estas piezas:

| Tema | Estado | Referencia |
| --- | --- | --- |
| Activación | Las trazas dependen de `CORE_DEBUG` y `BFF_DEBUG`, así que fuera del modo debug no hay trazas. | `internal/platform/telemetry/telemetry.go` (BFF y Core) |
| Destino | Las aplicaciones exportan directo a Jaeger, sin Collector. | `docker-compose.yaml` |
| Almacenamiento | Jaeger usa la imagen `latest`, almacenamiento en memoria y UI en `127.0.0.1`; las trazas se pierden al reiniciar. | `docker-compose.yaml`, servicio `jaeger` |
| Envío | El BFF descarta los envíos fallidos sin reintento ni registro. | `LOS_BFF/internal/platform/telemetry/telemetry.go` |
| Recurso | Solo `service.name`: no se puede filtrar por versión desplegada ni por ambiente. | `telemetry.go` (BFF y Core) |
| Correlación | `X-Request-ID` es un UUID de auditoría distinto del `trace_id`, y el `trace_id` no se devuelve al cliente. | `LOS_BFF/internal/platform/securityaudit/audit.go` |
| Logs | Texto plano en stdout, sin `trace_id` salvo en los logs de debug de autenticación. | `LOS_BFF/cmd/root.go`, `internal/auth/debug.go` |
| Métricas | No se emiten; no hay base para alertas ni SLO. | `go.mod` (BFF y Core) |
| Frontend | No envía `traceparent` ni muestra un código de soporte. | `LOS_FRONTEND/src/lib/api.ts` |
| Causa del error | Los spans marcan el código HTTP o gRPC, sin una causa segura (`error.type`). | Middleware HTTP e interceptores gRPC |

## Arquitectura propuesta

```text
Navegador ──traceparent──▶ BFF ──traceparent (metadata gRPC)──▶ Core ──▶ Postgres / Redis
                           │                                     │
                           └──────── OTLP: trazas, métricas, logs ┘
                                              │
                                              ▼
                                   OpenTelemetry Collector
                 memory_limiter → resource → redacción → tail_sampling → batch
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    ▼                         ▼                         ▼
          Jaeger (persistente)       Prometheus (métricas)         Loki (logs)
                    └──────────────── Grafana: enlace por trace_id ──────┘
```

El Collector separa las aplicaciones de los backends de telemetría:

| Capacidad | Sin Collector | Con Collector |
| --- | --- | --- |
| Conservar todas las trazas con error y muestrear el resto | No es posible: el muestreo se decide al iniciar la petición. | `tail_sampling` decide con la traza completa. |
| Eliminar datos sensibles | Depende de cada servicio. | Regla central adicional a la de cada servicio. |
| Cambiar de backend (Jaeger, Tempo, servicio administrado) | Cambio en cada servicio. | Cambio de un exportador en un archivo. |
| Caída del backend de telemetría | Cada exportador la maneja a su manera. | El Collector absorbe con buffer y reintentos. |

### Configuración del Collector

Distribución `otel/opentelemetry-collector-contrib` con versión fija.

```yaml
receivers:
  otlp:
    protocols:
      grpc: { endpoint: 0.0.0.0:4317 }
      http: { endpoint: 0.0.0.0:4318 }

processors:
  memory_limiter: { check_interval: 1s, limit_percentage: 80, spike_limit_percentage: 20 }
  resource:
    attributes:
      - { key: deployment.environment.name, value: "${env:LOS_ENVIRONMENT}", action: upsert }
  attributes/redact:
    actions:
      - { pattern: "(?i).*(token|password|secret|authorization|cookie|document).*", action: delete }
  tail_sampling:
    decision_wait: 10s
    policies:
      - { name: errores, type: status_code, status_code: { status_codes: [ERROR] } }
      - { name: lentas, type: latency, latency: { threshold_ms: 1000 } }
      - { name: muestra, type: probabilistic, probabilistic: { sampling_percentage: 10 } }
  batch: {}

exporters:
  otlp/jaeger:
    endpoint: jaeger:4317
    tls: { insecure: true }
  prometheus:
    endpoint: 0.0.0.0:8889
  otlphttp/loki:
    endpoint: http://loki:3100/otlp

service:
  pipelines:
    traces:  { receivers: [otlp], processors: [memory_limiter, resource, attributes/redact, tail_sampling, batch], exporters: [otlp/jaeger] }
    metrics: { receivers: [otlp], processors: [memory_limiter, resource, batch], exporters: [prometheus] }
    logs:    { receivers: [otlp], processors: [memory_limiter, resource, attributes/redact, batch], exporters: [otlphttp/loki] }
```

### Jaeger

- Versión fija de la imagen (`jaegertracing/jaeger:2.x.y`).
- Almacenamiento persistente: Badger en un nodo; OpenSearch o Elasticsearch en producción, con retención definida.
- UI publicada detrás del SSO de Keycloak para el equipo y soporte.
- Las aplicaciones envían al Collector (`OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317`), no a Jaeger.

## OpenTelemetry en cada servicio

### Configuración común (BFF y Core)

- La telemetría se activa con el endpoint OTLP y se puede apagar con `LOS_TELEMETRY_ENABLED=false`. `*_DEBUG` solo controla logs verbosos.
- Variables estándar: `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME`, `OTEL_RESOURCE_ATTRIBUTES`, `OTEL_TRACES_SAMPLER=parentbased_always_on` (el muestreo final lo hace el Collector).
- Recurso con `service.name`, `service.version` (tag de la imagen) y `deployment.environment.name`.
- Propagador global solo con `TraceContext`, sin `baggage`.
- Exportadores con reintento y un contador de envíos fallidos.

```go
res, err := resource.New(ctx,
    resource.WithFromEnv(),
    resource.WithTelemetrySDK(),
    resource.WithAttributes(semconv.ServiceVersion(version)),
)
traces, err := otlptracegrpc.New(ctx)
metrics, err := otlpmetricgrpc.New(ctx)

otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithBatcher(traces)))
otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metrics))))
otel.SetTextMapPropagator(propagation.TraceContext{})
```

### Trazas

| Capa | Práctica |
| --- | --- |
| HTTP (BFF) | Middleware con lista blanca de atributos (excluye `code` y `state` del callback OAuth) y header `X-Trace-Id` en todas las respuestas. |
| gRPC | Interceptores actuales, o `otelgrpc.NewClientHandler` / `NewServerHandler`, que además emiten métricas `rpc.*`. |
| Postgres | `otelpgx` sin parámetros de consulta (sin `WithIncludeQueryParameters`). |
| Redis | `redisotel` sin argumentos de comandos. |
| Errores | `span.SetStatus(codes.Error, ...)` en `5xx` y `error.type` con el código público o el `reason` de dominio; nunca el mensaje crudo. |
| Auditoría | El `trace_id` se usa como `CorrelationId` de los eventos de auditoría. |

### Métricas

- Automáticas: `http.server.request.duration` por `http.route` y `http.response.status_code`, `rpc.server.duration` y métricas del runtime de Go.
- De negocio: solicitudes creadas, avanzadas y rechazadas por fase, y conflictos de revisión.
- Alertas iniciales: tasa de `5xx` mayor a 1 % en 5 minutos, p95 de `/api/*` mayor a 1 s y servicios sin enviar telemetría.

### Logs

- `log/slog` con salida JSON.
- `trace_id` y `span_id` en cada log de una petición, mediante el puente `otelslog`, con envío por OTLP al Collector.
- Las reglas de privacidad de cada servicio se mantienen y el Collector las refuerza con `attributes/redact`.

### Frontend

1. `apiFetch` genera un `traceparent` por petición, para que el `traceId` exista aunque la respuesta falle sin cuerpo.
2. `apiFetch` interpreta `problem+json` y lanza `ApiError` con `status`, `code` y `traceId` (ver [Contratos de API](./contratos/contratos-api.md)).
3. Las pantallas de error muestran el código de soporte con opción de copiarlo.
4. En una etapa posterior, `@opentelemetry/sdk-trace-web` con instrumentación de `fetch`, enviando al Collector a través del BFF.

## Recorrido de un caso de soporte

1. **Reporte:** "No carga la meta del mes, código `4bf92f35…0e4736`".
2. **Traza (Jaeger):** búsqueda por `traceId`. Se ve:
   - el span del navegador;
   - `GET /api/home/monthly-goal` en el BFF;
   - la llamada gRPC al Core;
   - una consulta a Postgres de 9 s con `error.type = DEADLINE_EXCEEDED`.
3. **Logs (Loki):** desde la traza se abren los logs de BFF y Core de esa petición, con versión y ambiente.
4. **Métricas (Grafana):** el p95 de esa consulta subió desde la versión `0.3.1` y afecta a todos los tenants.
5. **Alerta:** la regla de p95 avisó antes del reporte.

| Pregunta | Señal que la responde |
| --- | --- |
| ¿Qué le pasó a la petición de este usuario? | Traza por `traceId`. |
| ¿Qué registraron los servicios en esa petición? | Logs filtrados por `trace_id`. |
| ¿Afecta a todos o a un caso? | Métricas por ruta, versión y ambiente. |
| ¿Empezó con un despliegue? | `service.version` en las tres señales. |
| ¿Se detecta antes que el usuario? | Alertas sobre métricas. |

## Plan de adopción

| Fase | Entregable | Repositorios |
| --- | --- | --- |
| 1. Errores y correlación | `problem+json` con lista blanca en el BFF, `X-Trace-Id`, `ApiError` y código de soporte en el frontend. | BFF, frontend |
| 2. Collector | Collector en `docker-compose`, aplicaciones apuntando a él, Jaeger con versión fija y almacenamiento persistente, telemetría separada de `*_DEBUG`. | Infraestructura, BFF, Core |
| 3. Logs y métricas | `slog` JSON con `trace_id`, métricas HTTP, gRPC y de negocio, Prometheus, Loki, Grafana y alertas. | BFF, Core, infraestructura |
| 4. Frontend instrumentado | Trazas del navegador hacia el Collector. | Frontend, BFF |
