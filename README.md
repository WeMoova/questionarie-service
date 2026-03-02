# Questionnaire Service

Backend API para la plataforma WeMoova — gestión de cuestionarios NOM-035, empresas, asignaciones, respuestas, reportes y gamificación.

## Stack

- **Go 1.24** + Chi v5
- **MongoDB** — base de datos principal
- **FusionAuth** — autenticación JWT (JWKS)
- **MinIO** — almacenamiento de imágenes (opcional)
- **Docker** — multi-stage build

## Requisitos

- Go 1.24+
- MongoDB 5.0+
- FusionAuth configurado ([guía](docs/FUSIONAUTH_SETUP.md))

## Quick Start

```bash
# Clonar e instalar
git clone https://github.com/WeMoova/questionarie-service.git
cd questionarie-service
go mod download

# Configurar
cp .env.example .env
# Editar .env con tus valores

# Crear índices en MongoDB
mongosh $MONGODB_URI < scripts/init_mongodb_indexes.js

# Ejecutar
go run main.go
# → http://localhost:8080
```

## Variables de Entorno

```bash
# Server
PORT=8080
ENV=development

# MongoDB
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=questionarie_db
MONGODB_TIMEOUT=10s

# FusionAuth
FUSIONAUTH_URL=https://auth.wemoova.com

# CORS
CORS_ORIGINS=https://services.wemoova.com,https://qa.services.wemoova.com,http://localhost:3000,http://localhost:3001

# MinIO (opcional)
MINIO_ENDPOINT=
MINIO_ACCESS_KEY=
MINIO_SECRET_KEY=
MINIO_BUCKET=questionnaire-images
MINIO_PUBLIC_URL=
```

## Arquitectura

```
handlers/    → HTTP handlers (request/response)
services/    → Lógica de negocio
repository/  → Acceso a MongoDB
models/      → Structs (bson + json tags)
middleware/  → JWT auth + RBAC
utils/       → Helpers (ParseRequestBody, RespondWithSuccess, etc.)
storage/     → MinIO client
db/          → MongoDB connection
```

Patrón Clean Architecture:

```
HTTP Request → Handler → Service → Repository → MongoDB
                ↑
            Middleware (JWT + RBAC)
```

## Colecciones MongoDB

| Colección | Propósito |
|-----------|-----------|
| `companies` | Empresas con branding |
| `questionnaires` | Cuestionarios con preguntas embebidas |
| `company_questionnaires` | Asignación cuestionario → empresa (períodos, estado) |
| `user_questionnaire_assignments` | Asignación → usuario (respuestas embebidas) |
| `users_metadata` | Perfiles de usuario vinculados a empresas |
| `gamification_badges` | Definición de badges |
| `gamification_achievements` | Definición de logros |
| `gamification_point_rules` | Reglas de puntuación |
| `gamification_user_profiles` | Puntos y badges por usuario |

## Roles y Permisos

| Rol | Puede hacer |
|-----|-------------|
| `super_admin` | Todo: CRUD cuestionarios, empresas, usuarios. Acceso global |
| `company_admin` | Gestionar SU empresa: asignar cuestionarios, ver reportes |
| `supervisor` | Gestionar SU equipo: asignar, ver progreso |
| `employee` | Responder cuestionarios asignados, ver su progreso |

## API Endpoints

Base path: `/questionarie-service/api/v1`

### Health
```
GET  /questionarie-service/health   — Liveness
GET  /questionarie-service/ready    — Readiness (incluye DB)
```

### Cuestionarios (Super Admin)
```
GET    /questionnaires                              — Listar
GET    /questionnaires/{id}                         — Obtener (incluye preguntas)
POST   /questionnaires                              — Crear
PUT    /questionnaires/{id}                         — Actualizar
DELETE /questionnaires/{id}                         — Soft delete
POST   /questionnaires/{id}/duplicate               — Duplicar
POST   /questionnaires/import                       — Importar desde Excel
PATCH  /questionnaires/{id}/toggle-status           — Activar/desactivar
PUT    /questionnaires/{id}/evaluation-config       — Configurar evaluación

POST   /questionnaires/{id}/questions               — Agregar pregunta
PUT    /questionnaires/{id}/questions/{qid}         — Actualizar pregunta
DELETE /questionnaires/{id}/questions/{qid}          — Eliminar pregunta

POST   /questionnaires/{id}/sections                — Agregar sección
PUT    /questionnaires/{id}/sections/{sid}          — Actualizar sección
DELETE /questionnaires/{id}/sections/{sid}           — Eliminar sección
```

### Empresas (Super Admin)
```
GET    /companies                                   — Listar
GET    /companies/{id}                              — Obtener
POST   /companies                                   — Crear
PUT    /companies/{id}                              — Actualizar
DELETE /companies/{id}                              — Eliminar
GET    /my-company                                  — Mi empresa (Company Admin+)
GET    /public/company-branding/{slug}              — Branding público (sin auth)
```

### Cuestionarios por empresa
```
POST   /companies/{cid}/questionnaires              — Asignar a empresa (Super Admin)
GET    /companies/{cid}/questionnaires              — Listar (Company Admin+)
GET    /company-questionnaires/{id}                 — Obtener
PUT    /company-questionnaires/{id}                 — Actualizar
DELETE /company-questionnaires/{id}                 — Eliminar
POST   /company-questionnaires/{id}/activate        — Activar
POST   /company-questionnaires/{id}/pause           — Pausar
POST   /company-questionnaires/{id}/close           — Cerrar
```

### Asignaciones a usuarios (Supervisor+)
```
POST   /company-questionnaires/{cqid}/assignments   — Asignar a usuarios
GET    /company-questionnaires/{cqid}/assignments    — Listar
DELETE /company-questionnaires/{cqid}/assignments    — Cancelar todas
POST   /assignments/{id}/cancel                      — Cancelar individual
POST   /company-questionnaires/{cqid}/assign-all     — Asignar a toda la empresa
POST   /company-questionnaires/{cqid}/assign-department — Asignar por departamento
```

### Respuestas (Employee+)
```
GET    /my-assignments                               — Mis asignaciones
GET    /my-questionnaires                            — Mis cuestionarios
POST   /company-questionnaires/{id}/start            — Iniciar cuestionario
GET    /assignments/{id}                             — Detalle de asignación
GET    /assignments/{id}/questions                   — Preguntas de asignación
POST   /assignments/{id}/responses                   — Guardar respuesta
PUT    /assignments/{id}/responses                   — Guardar múltiples respuestas
POST   /assignments/{id}/submit                      — Enviar completado
```

### User Metadata (Super Admin)
```
POST   /users/metadata                               — Crear
GET    /users/metadata                               — Listar (Supervisor+)
GET    /users/metadata/{uid}                         — Obtener
PUT    /users/metadata/{uid}                         — Actualizar
DELETE /users/metadata/{uid}                         — Eliminar
GET    /users/me/metadata                            — Mi metadata (todos)
GET    /companies/{cid}/users                        — Usuarios de empresa
POST   /users/metadata/resolve-documents             — Resolver por documento
```

### Reportes (Supervisor+)
```
GET    /reports/company-questionnaire/{cqid}/completion       — Completitud
GET    /reports/company/{cid}/overview                        — Overview empresa
GET    /reports/company/{cid}/employees-progress              — Progreso empleados
GET    /reports/assignments/{id}                              — Reporte individual
GET    /reports/company-questionnaire/{cqid}/answers          — Distribución respuestas
GET    /reports/company-questionnaire/{cqid}/evaluation-summary — Evaluación
GET    /reports/company/{cid}/trends                          — Tendencias
GET    /reports/company-questionnaire/{cqid}/export           — Exportar CSV
```

### Categorías (Super Admin)
```
POST   /questionnaire-categories                     — Crear
GET    /questionnaire-categories                     — Listar
GET    /questionnaire-categories/{id}                — Obtener
PUT    /questionnaire-categories/{id}                — Actualizar
DELETE /questionnaire-categories/{id}                — Eliminar
GET    /questionnaire-categories/{id}/questionnaires — Cuestionarios de categoría
```

### Gamificación
```
# Admin (Super Admin)
POST   /gamification/badges                          — Crear badge
GET    /gamification/badges                          — Listar badges
PUT    /gamification/badges/{id}                     — Actualizar badge
POST   /gamification/achievements                    — Crear logro
GET    /gamification/achievements                    — Listar logros
PUT    /gamification/achievements/{id}               — Actualizar logro
GET    /gamification/point-rules                     — Listar reglas
PUT    /gamification/point-rules/{id}                — Actualizar regla

# Usuario (Employee+)
GET    /gamification/my-profile                      — Mi perfil gamificado
GET    /gamification/leaderboard                     — Leaderboard global

# Empresa (Company Admin+)
GET    /gamification/company/{cid}/leaderboard       — Leaderboard empresa
GET    /gamification/users/{uid}/profile             — Perfil de usuario
```

### Imágenes
```
POST   /images/upload                                — Subir imagen (Super Admin)
GET    /images/*                                     — Servir imagen (público)
DELETE /images                                       — Eliminar imagen (Super Admin)
```

### Progreso y visibilidad (Supervisor+)
```
GET    /company-questionnaires/{cqid}/progress         — Progreso
GET    /company-questionnaires/{cqid}/pending-users    — Usuarios pendientes
GET    /company-questionnaires/{cqid}/in-progress-users — En progreso
GET    /company-questionnaires/{cqid}/completed-users   — Completados
POST   /company-questionnaires/{cqid}/remind            — Enviar recordatorio
```

### Dashboard
```
GET    /companies/{cid}/dashboard                      — Dashboard empresa
GET    /questionnaires/{id}/stats                      — Stats de cuestionario
GET    /questionnaires/{id}/companies                  — Empresas usando cuestionario
GET    /my-company/questionnaires                      — Cuestionarios de mi empresa
GET    /my-team/assignments                            — Asignaciones de mi equipo
```

## Swagger UI

```
http://localhost:8080/questionarie-service/swagger/
```

Documentación interactiva de todos los endpoints. Usar "Authorize" con `Bearer {jwt-token}`.

## Docker

```bash
docker build -t questionarie-service .
docker run -p 8080:8080 \
  -e MONGODB_URI=mongodb://host:27017 \
  -e MONGODB_DATABASE=questionarie_db \
  -e FUSIONAUTH_URL=https://auth.wemoova.com \
  questionarie-service
```

Multi-stage build: `golang:1.24-alpine` → `alpine:latest`. Corre como usuario no-root.

## Make

```bash
make build          # Compilar
make run            # Ejecutar
make test           # Tests con coverage
make docker-build   # Build Docker
make deps           # Descargar dependencias
```

## CI/CD

- Push a `main` → deploy a QA automáticamente
- Release (tag) → deploy a producción
- Pipeline: build Docker → push a GHCR → actualizar manifiesto en argo-apps → ArgoCD sync

## Documentación

- [FusionAuth Setup](docs/FUSIONAUTH_SETUP.md)
- [API Examples](docs/API_EXAMPLES.md)
- [Postman Collection](postman_collection.json)
- [OpenAPI Spec](docs/swagger.json)

## Dominios

| Entorno | URL |
|---------|-----|
| QA | `https://qa.services.wemoova.com/questionarie-service` |
| Producción | `https://services.wemoova.com/questionarie-service` |
