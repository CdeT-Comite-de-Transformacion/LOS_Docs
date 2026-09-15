# Reglas de desarrollo
## Nomenclaturas
Para mantener la consistencia y legibilidad en todo el proyecto, todas las convenciones de nombres siguen las siguientes estructuras:
#### kebab-case
* **Archivos y carpetas:** `component`,`user-component.js`, `main-styles.css`, `main.go`
* **Rutas / URLs:** `/user-profile`, `/api/v1/user-profile`
* **Clases y selectores CSS:** `.main-button`, `.card-container`
* **Ramas de Git:** `feature/new-window`, `bugfix/error-login`
#### PascalCase
* **Clases:** `NewClass()`
* **Interfases:** `NewInterface()`
#### camelCase
* **Métodos y funciones:** `getUserById()`, `calculateTotal()`
* **Variables y parámetros:** `let userList = []`, `var totalAmount = 10`
* **Propiedades de objetos / JSON:** `{ "firstName": "John", "isActive": true }`
#### snake_case
* **Bases de datos:** `app_db`, `metrics_prod`
* **Tablas:** `users`, `user_profiles`, `order_items`
* **Campos / Columnas:** `id`, `user_id`, `first_name`, `created_at`, `is_active`
>#### [Guía nomenclaturas Go](https://www.alexedwards.net/blog/go-naming-conventions)
## Idioma
Todo el proyecto (código fuente, comentarios, commits, documentación y nombres de variables) debe escribirse **exclusivamente en inglés**. 
* **Comentarios y documentación:** `// Fetch user data from the database`
* **Mensajes de commit:** `git commit -m "feat: add user authentication"`
* **Nombres de variables y funciones:** `const userList = []`, `func getUserById()`

> **Nota:** Se debe evitar el uso de español o spanglish en cualquier parte del código del proyecto.
## Documentación de código
Todo el código debe documentarse siguiendo las convenciones de cada lenguaje antes de ser integrado:
#### TypeScript / JavaScript (JSDoc)
Usa bloques `/** ... */` sobre funciones, clases e interfaces especificando parámetros (`@param`), valores de retorno (`@returns`) y tipos.
```
typescript
/**
 * Calculates the total price including tax.
 *
 * @param amount - The base amount before tax.
 * @param taxRate - The tax percentage to apply.
 * @returns The final total price.
 */
function calculateTotal(amount: number, taxRate: number): number {
  return amount + (amount * taxRate);
}
```
[Documentación TypeScript](https://www.typescriptlang.org/docs/handbook/jsdoc-supported-types.html)
>
#### Go (GoDoc)
Escribe comentarios simples de línea // directamente sobre declaraciones exportadas (tipos, funciones, constantes o paquetes). Todo comentario debe comenzar con el mismo nombre de la entidad que documenta.
```
// User represents an authenticated account in the system.
type User struct {
    ID   string
    Name string
}

// GetUserByID retrieves a user record matching the provided ID.
// It returns an error if the user cannot be found.
func GetUserByID(id string) (*User, error) {
    // ...
}
```
[Documentación Go](https://go.dev/src/go/doc/example.go)
