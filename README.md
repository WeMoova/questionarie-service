# Questionnaire Service - NOM-035 Management System

Sistema completo de gestión de cuestionarios inspirado en NOM-035 (Norma Mexicana de Riesgos Psicosociales) con soporte para múltiples tipos de preguntas, asignación jerárquica por roles, y reportes agregados por empresa.

## 📋 Tabla de Contenidos

- [Características](#características)
- [Arquitectura](#arquitectura)
- [Requisitos](#requisitos)
- [Instalación](#instalación)
- [Configuración](#configuración)
- [Roles y Permisos](#roles-y-permisos)
- [API Endpoints](#api-endpoints)
- [Modelos de Datos](#modelos-de-datos)
- [Flujo de Uso](#flujo-de-uso)
- [Deployment](#deployment)

## ✨ Características

### Gestión de Cuestionarios
- ✅ Creación de cuestionarios con múltiples tipos de preguntas
- ✅ Tipos de preguntas: Opción múltiple, Escala Likert, Texto libre, Sí/No
- ✅ Activación/desactivación de cuestionarios
- ✅ Gestión de preguntas embebidas (CRUD completo)

### Gestión de Empresas
- ✅ CRUD de empresas
- ✅ Asignación de cuestionarios a empresas con períodos definidos
- ✅ Gestión de períodos de respuesta

### Gestión de Usuarios
- ✅ Autenticación 100% via FusionAuth
- ✅ 4 niveles de roles: Super Admin, Company Admin, Supervisor, Employee
- ✅ Metadata de usuarios vinculada a empresas
- ✅ Jerarquía de supervisores

### Asignaciones
- ✅ Asignación de cuestionarios a empleados
- ✅ Validación de períodos activos
- ✅ Estados: Pendiente, En Progreso, Completado
- ✅ Prevención de asignaciones duplicadas

### Respuestas
- ✅ Guardado incremental de respuestas
- ✅ Validación de preguntas requeridas
- ✅ Respuestas embebidas en asignaciones
- ✅ Historial completo

### Reportes y Métricas
- ✅ Reportes agregados por empresa (sin datos individuales)
- ✅ Métricas de completitud detalladas
- ✅ Estadísticas por departamento
- ✅ Tiempo promedio de completitud
- ✅ Overview de empresa con todos los cuestionarios

## 🏗 Arquitectura

### Stack Tecnológico
- **Lenguaje**: Go 1.21+
- **Framework Web**: Chi v5
- **Base de Datos**: MongoDB 5.0+
- **Autenticación**: FusionAuth (JWT con JWKS)
- **Deployment**: Docker + Kubernetes

### Patrón de Diseño
```
Clean Architecture con capas separadas:

┌─────────────────────────────────────┐
│         HTTP Handlers               │  ← Entrada HTTP
├─────────────────────────────────────┤
│      Middleware (JWT, RBAC)         │  ← Autenticación/Autorización
├─────────────────────────────────────┤
│         Services                     │  ← Lógica de negocio
├─────────────────────────────────────┤
│         Repositories                 │  ← Acceso a datos
├─────────────────────────────────────┤
│         MongoDB                      │  ← Persistencia
└─────────────────────────────────────┘
```

### Modelo de Datos MongoDB

**Colecciones:**
- `companies` - Empresas
- `questionnaires` - Cuestionarios con preguntas embebidas
- `company_questionnaires` - Asignaciones de cuestionarios a empresas
- `user_questionnaire_assignments` - Asignaciones a usuarios con respuestas embebidas
- `users_metadata` - Metadata de usuarios (vinculación con empresas)

**Ventajas del diseño:**
- Preguntas embebidas → 1 consulta en vez de JOINs
- Respuestas embebidas → Histórico completo sin fragmentación
- Esquema flexible para diferentes tipos de preguntas
- Agregaciones nativas de MongoDB para reportes

## 📦 Requisitos

- Go 1.21 o superior
- MongoDB 5.0 o superior
- FusionAuth configurado (ver [FUSIONAUTH_SETUP.md](docs/FUSIONAUTH_SETUP.md))
- Docker (opcional, para deployment)

## 🚀 Instalación

### 1. Clonar el repositorio
```bash
git clone <repository-url>
cd questionarie-service
```

### 2. Instalar dependencias
```bash
go mod download
```

### 3. Configurar variables de entorno
```bash
cp .env.example .env
# Editar .env con tus valores
```

### 4. Crear índices en MongoDB
```bash
mongosh <MONGODB_URI> < scripts/init_mongodb_indexes.js
```

### 5. Ejecutar el servicio
```bash
go run main.go
```

El servicio estará disponible en `http://localhost:8080`

## ⚙️ Configuración

### Variables de Entorno

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
CORS_ORIGINS=*
```

### Configuración de FusionAuth

Ver guía completa en [docs/FUSIONAUTH_SETUP.md](docs/FUSIONAUTH_SETUP.md)

**Resumen:**
1. Crear aplicación en FusionAuth
2. Configurar roles: `super_admin`, `company_admin`, `supervisor`, `employee`
3. Configurar JWT issuer y audience
4. Obtener JWKS endpoint

## 👥 Roles y Permisos

### Super Admin (`super_admin`)
- ✅ Crear/editar/desactivar cuestionarios
- ✅ Gestionar preguntas
- ✅ Crear/editar empresas
- ✅ Asignar cuestionarios a empresas
- ✅ Crear/editar user metadata
- ✅ Acceso a todos los reportes

### Company Admin (`company_admin`)
- ✅ Ver cuestionarios asignados a SU empresa
- ✅ Asignar cuestionarios a empleados de SU empresa
- ✅ Ver reportes de SU empresa
- ❌ No puede ver otras empresas

### Supervisor (`supervisor`)
- ✅ Ver cuestionarios de su empresa
- ✅ Asignar cuestionarios a SU equipo
- ✅ Ver progreso de SU equipo
- ✅ Ver reportes de su equipo
- ❌ No puede asignar a empleados de otros supervisores

### Employee (`employee`)
- ✅ Ver cuestionarios asignados a SÍ MISMO
- ✅ Responder cuestionarios
- ✅ Ver su propio progreso
- ❌ No puede ver respuestas de otros

## 🔌 API Endpoints

### Swagger UI Documentation

El servicio incluye **Swagger UI** para explorar y probar todos los endpoints de forma interactiva:

```
🌐 Swagger UI: http://localhost:8080/questionarie-service/swagger/
📄 OpenAPI JSON: http://localhost:8080/questionarie-service/swagger/doc.json
```

**Características de Swagger UI:**
- ✅ Documentación interactiva de todos los endpoints
- ✅ Prueba de endpoints directamente desde el navegador
- ✅ Autenticación con token JWT (botón "Authorize")
- ✅ Ejemplos de request/response para cada endpoint
- ✅ Filtrado por tags (Questionnaires, Companies, Assignments, Reports, etc.)

**Cómo usar Swagger UI:**
1. Inicia el servicio: `go run main.go`
2. Abre en tu navegador: `http://localhost:8080/questionarie-service/swagger/`
3. Haz clic en "Authorize" e ingresa: `Bearer {tu-jwt-token}`
4. Explora y prueba los endpoints

### Health Checks
```
GET  /questionarie-service/health        - Health check
GET  /questionarie-service/ready         - Readiness check (incluye MongoDB)
```

### Questionnaires (Super Admin)
```
POST   /api/v1/questionnaires                           - Crear cuestionario
GET    /api/v1/questionnaires                           - Listar cuestionarios
GET    /api/v1/questionnaires/:id                       - Obtener cuestionario
PUT    /api/v1/questionnaires/:id                       - Actualizar cuestionario
DELETE /api/v1/questionnaires/:id                       - Desactivar cuestionario

POST   /api/v1/questionnaires/:id/questions             - Agregar pregunta
PUT    /api/v1/questionnaires/:id/questions/:question_id - Actualizar pregunta
DELETE /api/v1/questionnaires/:id/questions/:question_id - Eliminar pregunta
```

### Companies (Super Admin)
```
POST   /api/v1/companies                  - Crear empresa
GET    /api/v1/companies                  - Listar empresas
GET    /api/v1/companies/:id              - Obtener empresa
PUT    /api/v1/companies/:id              - Actualizar empresa

POST   /api/v1/companies/:company_id/questionnaires  - Asignar cuestionario a empresa
GET    /api/v1/companies/:company_id/questionnaires  - Listar cuestionarios de empresa
```

### User Metadata (Super Admin)
```
POST   /api/v1/users/metadata              - Crear metadata de usuario
GET    /api/v1/users/metadata/:user_id     - Obtener metadata
PUT    /api/v1/users/metadata/:user_id     - Actualizar metadata
DELETE /api/v1/users/metadata/:user_id     - Eliminar metadata

GET    /api/v1/companies/:company_id/users - Listar usuarios de empresa
```

### Assignments (Company Admin, Supervisor)
```
POST   /api/v1/company-questionnaires/:cq_id/assignments  - Asignar a usuarios
GET    /api/v1/company-questionnaires/:cq_id/assignments  - Listar asignaciones
GET    /api/v1/my-company/questionnaires                  - Cuestionarios de mi empresa
GET    /api/v1/my-team/assignments                        - Asignaciones de mi equipo
```

### Responses (Employee)
```
GET    /api/v1/my-assignments               - Mis cuestionarios asignados
GET    /api/v1/assignments/:id              - Detalle de asignación

POST   /api/v1/assignments/:id/responses    - Guardar respuesta
PUT    /api/v1/assignments/:id/responses    - Actualizar múltiples respuestas
POST   /api/v1/assignments/:id/submit       - Enviar cuestionario completado
```

### Reports (Company Admin, Supervisor)
```
GET    /api/v1/reports/company-questionnaire/:cq_id/completion  - Métricas de completitud
GET    /api/v1/reports/company/:company_id/overview             - Overview de empresa
GET    /api/v1/reports/company/:company_id/employees-progress   - Progreso de empleados
```

## 📚 Documentación Adicional

- [FusionAuth Setup Guide](docs/FUSIONAUTH_SETUP.md) - Configuración de autenticación
- [API Examples](docs/API_EXAMPLES.md) - Ejemplos completos de uso
- [Postman Collection](postman_collection.json) - Collection para testing

## 🐳 Deployment

### Docker
```bash
docker build -t questionarie-service .
docker run -p 8080:8080 \
  -e MONGODB_URI=mongodb://host:27017 \
  -e MONGODB_DATABASE=questionarie_db \
  -e FUSIONAUTH_URL=https://auth.wemoova.com \
  questionarie-service
```

## 🧪 Testing

```bash
# Unit tests
go test ./...

# Con coverage
go test -cover ./...
```

## 📝 License

This project is licensed under the MIT License.

---

**Generado con** [Claude Code](https://claude.com/claude-code)
