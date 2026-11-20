---
sidebar_position: 1
title: Contratos de API
hide_title: true
---

<h1 align="center">Contratos de API</h1>

<div align="center">
  <img src="https://img.shields.io/badge/OpenAPI_3.1-6BA539?style=for-the-badge&logo=openapiinitiative&logoColor=white" alt="OpenAPI 3.1"/>
  <img src="https://img.shields.io/badge/RFC_9457-4A4A55?style=for-the-badge" alt="RFC 9457"/>
  <img src="https://img.shields.io/badge/W3C_Trace_Context-005A9C?style=for-the-badge" alt="W3C Trace Context"/>
  <img src="https://img.shields.io/badge/JSON-000000?style=for-the-badge&logo=json&logoColor=white" alt="JSON"/>
  <img src="https://img.shields.io/badge/ISO_8601_%C2%B7_ISO_4217-2E7D32?style=for-the-badge" alt="ISO 8601 · ISO 4217"/>
</div>

<p align="center">
  <i>Formato común de las respuestas del BFF y proceso para acordar nuevos contratos.</i>
</p>
<p align="center">
  <i><b>Estado: propuesta para revisión</b></i>
</p>
<br />

---

Formato común para las respuestas HTTP que el BFF entrega al frontend. El objetivo es que todos los endpoints se lean igual, que cada dato tenga un solo lugar y que cada error se pueda rastrear con un identificador de correlación.

## Estándares de referencia

| Necesidad | Estándar |
| --- | --- |
| Definición del contrato | OpenAPI 3.1 |
| Formato de errores | RFC 9457, Problem Details for HTTP APIs |
| Identificador de correlación | W3C Trace Context (`traceparent`) |
| Fechas y periodos | ISO 8601 |
| Moneda | ISO 4217 |

## Envoltura de la respuesta

Toda respuesta `2xx` con cuerpo tiene dos bloques:

```json
{
  "data": {},
  "meta": {
    "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
    "generatedAt": "2026-11-20T15:30:00Z"
  }
}
```

| Bloque | Contenido |
| --- | --- |
| `data` | La información que la pantalla necesita, organizada en las secciones de la tabla siguiente. |
| `meta` | Datos de la respuesta, no del negocio: `traceId` (siempre), `generatedAt` (siempre) y `page` en colecciones paginadas (`{ "size": 50, "nextCursor": "..." }`). |

Los eventos SSE usan la misma envoltura en el campo `data:` de cada evento.

## Secciones de `data`

`data` se organiza en secciones con nombre fijo. Un endpoint incluye solo las que aplican.

| Sección | Contenido | Ejemplo de uso |
| --- | --- | --- |
| `workflow` | Identidad del flujo: `id`, `name`, `version`. | Respuestas que dependen de una versión de flujo. |
| `phase` | Fase actual: `code`, `name`, `position`, `version`. | Contrato de formulario de una fase. |
| `request` | Identidad de la solicitud: `id`, `revision`. | Cualquier respuesta sobre una solicitud. |
| Recurso principal | Los datos de negocio, con un nombre que los describe (`stages`, `deals`, `goal`). | Todas. |
| `ui` | Contrato de presentación para formularios dinámicos: `contractVersion`, `schema`, `uiSchema`, `behavior`. | Solo endpoints de formulario. |
| `values` | Valores actuales del formulario, con los tipos que define `ui.schema`. | Solo endpoints de formulario. |

Reglas de organización:

- Los datos de negocio nunca se mezclan dentro de `ui`, y `ui` no contiene datos del usuario.
- Cuando una respuesta agrupa datos de varios flujos, cada grupo repite la misma forma: `{ "workflow": {...}, "<recurso>": [...] }`.
- `mappings` del contrato de fase es de uso interno del backend y no se incluye en la respuesta.

## Ejemplos

### Resumen por fase

```json
{
  "data": {
    "workflows": [
      {
        "workflow": {
          "id": "d4c36c24-21c1-4c9d-a85a-1e27ed2b6b5f",
          "name": "Crédito vehicular · Perú",
          "version": 1,
          "release": "25.1.0"
        },
        "stages": [
          { "code": "intake", "name": "Radicación", "position": 1, "requests": 40, "value": { "amount": 1996000, "currency": "PEN" } },
          { "code": "document_upload", "name": "Cargue documental", "position": 2, "requests": 32, "value": { "amount": 1602000, "currency": "PEN" } }
        ]
      }
    ]
  },
  "meta": { "traceId": "4bf92f3577b34da6a3ce929d0e0e4736", "generatedAt": "2026-11-20T15:30:00Z" }
}
```

Correspondencia con la respuesta actual de `/api/requests/summary`:

| Campo actual | Campo propuesto | Nota |
| --- | --- | --- |
| `workflowId`, `workflowName`, `workflowVersion` | `workflow.id`, `workflow.name`, `workflow.version` | La versión publicada (`v25.1.0`) pasa a `workflow.release`, fuera del nombre. |
| `stageCode`, `displayName` | `code`, `name` | Mismo par código + nombre en todos los catálogos. |
| `count` | `requests` | |
| `value` | `value.amount` + `value.currency` | |
| `totalRequests`, `percentage` | Se calculan en el frontend | Se deducen de `requests`. |
| `href`, `icon` | Se resuelven en el frontend por `code` | Rutas e íconos son decisiones de la interfaz. |

### Contrato de formulario de una fase

```json
{
  "data": {
    "workflow": { "id": "d4c36c24-21c1-4c9d-a85a-1e27ed2b6b5f", "name": "Crédito vehicular · Perú", "version": 1 },
    "phase": { "code": "intake", "name": "Radicación", "position": 1, "version": "stage1.1" },
    "request": { "id": "req-1", "revision": 3 },
    "ui": {
      "contractVersion": "1.0",
      "schema": {},
      "uiSchema": {},
      "behavior": {}
    },
    "values": { "ingresos": 8000, "tieneAvalista": false }
  },
  "meta": { "traceId": "4bf92f3577b34da6a3ce929d0e0e4736", "generatedAt": "2026-11-20T15:30:00Z" }
}
```

### Meta del mes

```json
{
  "data": {
    "goal": {
      "period": "2026-11",
      "target": { "amount": 4000000, "currency": "PEN" },
      "disbursed": { "amount": 2720000, "currency": "PEN" }
    },
    "topDealers": [
      { "id": "d-01", "name": "Autoplaza del Norte", "filings": 24 }
    ]
  },
  "meta": { "traceId": "4bf92f3577b34da6a3ce929d0e0e4736", "generatedAt": "2026-11-20T15:30:00Z" }
}
```

## Reglas de campos

| Tema | Regla |
| --- | --- |
| Nombres | camelCase, en inglés, sustantivos que describen el dato (`requests`, no `count`). |
| Catálogos | Siempre `{ "code", "name" }`. Un concepto usa el mismo código en todos los endpoints (la fase de radicación es `intake` en el resumen y en el formulario). |
| Montos | `{ "amount", "currency" }`, con 2 decimales como máximo y moneda ISO 4217. |
| Fechas | ISO 8601 en UTC (`2026-11-20T15:30:00Z`); periodos como `"2026-11"`. |
| Versiones | Cada dato en su campo: el nombre de un flujo no incluye su versión. |
| Valores derivados | No se envían si se deducen de otros campos de la misma respuesta (totales, porcentajes, restantes). |
| Interfaz | El backend no envía rutas, íconos ni estilos fuera de la sección `ui`. |
| Colecciones | Una colección vacía es `[]`, nunca `null`. |
| Ausencia de valor | El campo se omite; `null` solo cuando "sin valor" es un dato del negocio. |
| Alcance | El usuario y el tenant se obtienen de la sesión, no de parámetros enviados por el frontend. |
| Consumidores | Si dos pantallas necesitan datos distintos, se exponen dos endpoints. |

## Errores

### Separación por audiencia

La respuesta HTTP es visible en la pestaña Network del navegador, por lo que solo contiene lo que el usuario puede ver: un código público, un texto seguro y el `traceId`. La causa técnica se registra en trazas y logs, y el equipo la consulta con ese `traceId` (ver [Observabilidad](../observabilidad.md)).

| Audiencia | Recibe | Dónde |
| --- | --- | --- |
| Usuario | Qué falló, qué puede hacer y un código de soporte. | Pantalla y respuesta HTTP. |
| Equipo | Servicio, dependencia, consulta y contexto de la falla. | Trazas, logs y métricas filtrados por `traceId`. |

La respuesta nunca incluye mensajes internos, códigos gRPC, nombres de servicios, SQL, rutas de archivos ni stack traces.

### Formato

`Content-Type: application/problem+json`

```json
{
  "type": "/errores/request-revision-conflict",
  "title": "La solicitud cambió en otra sesión",
  "status": 409,
  "code": "REQUEST_REVISION_CONFLICT",
  "traceId": "4bf92f3577b34da6a3ce929d0e0e4736"
}
```

| Campo | Regla |
| --- | --- |
| `type` | URI relativa a la entrada del catálogo de errores. |
| `title` | Texto en español apto para el usuario. El frontend puede reemplazarlo según `code`. |
| `status` | Igual al código HTTP. |
| `code` | Código estable. Es el único campo con el que el frontend decide su comportamiento. |
| `traceId` | Identificador de la traza de la petición. |
| `errors` | Solo en `422`: lista de `{ "pointer": "/applicant/document", "code": "REQUIRED" }` con JSON Pointer al campo. |

Los errores `5xx` usan siempre un texto genérico y los códigos `SERVICE_UNAVAILABLE` o `INTERNAL`.

### Catálogo inicial

| HTTP | `code` | Uso |
| --- | --- | --- |
| 400 | `INVALID_REQUEST` | Cuerpo o parámetros mal formados. |
| 401 | `SESSION_EXPIRED` | Sesión vencida; el frontend lleva al login. |
| 403 | `FORBIDDEN` | Sin permiso; el frontend recarga los permisos. |
| 404 | `NOT_FOUND` | El recurso no existe o el usuario no tiene acceso; ambos casos responden igual. |
| 409 | `REQUEST_REVISION_CONFLICT` | La solicitud cambió en otra sesión. |
| 422 | `VALIDATION_FAILED` | Reglas de negocio o de schema; incluye `errors`. |
| 429 | `RATE_LIMITED` | Demasiadas peticiones. |
| 503 | `SERVICE_UNAVAILABLE` | Core, base de datos o proveedor de identidad no disponibles. |
| 500 | `INTERNAL` | Cualquier otra falla. |

### Recorrido de un error

```text
Core (gRPC)                    BFF                                    Navegador
status + ErrorInfo.reason ───▶ reason → code (lista blanca) ─────────▶ problem+json
mensaje interno           ───▶ solo traza y log, nunca el cuerpo
```

1. **Core:** responde el status gRPC con `google.rpc.ErrorInfo`, donde `reason` es el código de dominio. El mensaje del status es solo para telemetría.
2. **BFF:** traduce `reason` a `code` con una lista blanca, registra la causa en la traza (`error.type`) y responde `problem+json`. Un `reason` desconocido se responde como `INTERNAL`.
3. **Frontend:** decide por `code`, muestra el texto de su catálogo (o `title`) y muestra el `traceId` como código de soporte con opción de copiarlo.

```text
No pudimos cargar la meta del mes.            [Reintentar]
Código de soporte: 4bf92f35…0e4736  [Copiar]
```

## Nuevos contratos

Cada contrato entre frontend y backend se documenta en `documentation/projects/contratos/` de LOS_Docs, un archivo por dominio (`catalogos-vehiculos.md`, `home.md`), con la estructura de este documento:

| Sección | Contenido |
| --- | --- |
| Encabezado | Título, insignias, propósito en una línea y estado (`propuesta para revisión` o `aprobado`). |
| Objetivo | Pantalla o proceso que atiende y tarea que lo origina. |
| Endpoints | Método, ruta, parámetros y su descripción en OpenAPI 3.1. |
| Secciones de `data` | Secciones que usa la respuesta, entre las definidas en este documento. |
| Ejemplos | Una respuesta completa por endpoint, con envoltura y `meta`. |
| Reglas de campos | Solo las propias del dominio; las generales se toman de este documento. |
| Errores | Códigos `code` nuevos, con su estado HTTP y su uso, que se agregan al catálogo. |
| Por definir | Puntos abiertos, con su responsable. |

### Aprobación

1. Quien necesita el contrato, del frontend o del backend, lo redacta y abre un PR a LOS_Docs.
2. Los responsables de frontend y backend de la tarea lo revisan.
3. Con el PR aprobado, el contrato pasa a estado `aprobado` y se comparte con el responsable de backend, de frontend o de ambos, según quién lo solicitó.

### Vía ágil

Los responsables de la tarea pueden acordar implementar el contrato sin esperar el PR. En ese caso:

1. El contrato se registra en `docs/contratos/` del repositorio LOS_CORE, con la misma estructura.
2. Frontend y backend implementan sobre ese archivo.
3. El PR a LOS_Docs se abre después, con el contrato tal como quedó implementado.

## Versionado y validación

1. Spectral valida en CI las reglas de este documento: envoltura, `problem+json` y nombres.
2. El frontend genera sus tipos TypeScript desde OpenAPI (`openapi-typescript`).
3. Las pruebas del BFF validan cada respuesta contra su schema OpenAPI.
4. Agregar campos opcionales es compatible. Quitar o renombrar un campo requiere publicar el nuevo, marcar el anterior como `deprecated` y retirarlo cuando ningún cliente lo use.
