# FusionAuth Setup Guide - Questionnaire Service

Esta guía detalla cómo configurar FusionAuth para autenticar y autorizar usuarios en el servicio de cuestionarios NOM-035.

## 📋 Tabla de Contenidos

- [Requisitos Previos](#requisitos-previos)
- [Configuración de la Aplicación](#configuración-de-la-aplicación)
- [Configuración de Roles](#configuración-de-roles)
- [Configuración de JWT](#configuración-de-jwt)
- [Obtener JWKS Endpoint](#obtener-jwks-endpoint)
- [Testing de Autenticación](#testing-de-autenticación)
- [Mapeo de Claims](#mapeo-de-claims)

## Requisitos Previos

- Acceso a instancia de FusionAuth: `https://auth.wemoova.com`
- Permisos de administrador en FusionAuth
- Conocimiento básico de OAuth 2.0 y JWT

## Configuración de la Aplicación

### 1. Crear Aplicación en FusionAuth

1. Acceder a FusionAuth Admin Panel: `https://auth.wemoova.com/admin`
2. Navegar a **Applications** en el menú lateral
3. Hacer clic en el botón verde **+** (Add Application)
4. Configurar los siguientes campos:

**Pestaña OAuth:**
- **Name**: `Questionnaire Service`
- **Application Id**: Se genera automáticamente (copiar para uso posterior)
- **Tenant**: Seleccionar tenant apropiado (ej: `wemoova`)

**Authorized redirect URLs:**
```
http://localhost:8080/questionarie-service/auth/callback
https://qa.services.wemoova.com/questionarie-service/auth/callback
https://services.wemoova.com/questionarie-service/auth/callback
```

**Logout URL:**
```
https://qa.services.wemoova.com/questionarie-service/logout
```

**Enabled grants:**
- ✅ Authorization Code
- ✅ Refresh Token

5. Hacer clic en **Save**

### 2. Obtener Client ID y Client Secret

Después de crear la aplicación:

1. En la lista de aplicaciones, hacer clic en el ícono de **View** (ojo) junto a "Questionnaire Service"
2. Copiar los siguientes valores:
   - **Application Id** (Client ID)
   - **Client secret** (revelar y copiar)

Guardar estos valores de forma segura, se usarán en la configuración del servicio.

## Configuración de Roles

FusionAuth debe configurarse con 4 roles específicos para este servicio.

### 1. Crear Roles en la Aplicación

1. En FusionAuth Admin, ir a **Applications** → **Questionnaire Service**
2. Ir a la pestaña **Roles**
3. Hacer clic en **Add Role** y crear los siguientes roles:

#### Role 1: Super Admin
- **Name**: `super_admin`
- **Description**: `Super administrador con acceso total al sistema`
- **Is Super Role**: ❌ No
- **Is Default**: ❌ No

#### Role 2: Company Admin
- **Name**: `company_admin`
- **Description**: `Administrador de empresa con acceso limitado a su empresa`
- **Is Super Role**: ❌ No
- **Is Default**: ❌ No

#### Role 3: Supervisor
- **Name**: `supervisor`
- **Description**: `Supervisor con acceso a su equipo de trabajo`
- **Is Super Role**: ❌ No
- **Is Default**: ❌ No

#### Role 4: Employee
- **Name**: `employee`
- **Description**: `Empleado con acceso solo a sus cuestionarios asignados`
- **Is Super Role**: ❌ No
- **Is Default**: ✅ **Sí** (rol por defecto para nuevos usuarios)

4. Hacer clic en **Save** después de crear cada rol

### 2. Asignar Roles a Usuarios

Para asignar roles a usuarios existentes:

1. Ir a **Users** en el menú lateral
2. Buscar y seleccionar un usuario
3. Ir a la pestaña **Registrations**
4. Hacer clic en **Add Registration**
5. Seleccionar **Application**: `Questionnaire Service`
6. En **Roles**, seleccionar uno o más roles:
   - Para super admin: solo `super_admin`
   - Para admin de empresa: `company_admin` y `employee`
   - Para supervisor: `supervisor` y `employee`
   - Para empleado: solo `employee`
7. Hacer clic en **Save**

### 3. Configurar Roles en JWT Claims

Para que los roles aparezcan en el JWT token:

1. Ir a **Applications** → **Questionnaire Service**
2. Ir a la pestaña **JWT**
3. En **Enabled**, activar ✅ (si no está activado)
4. **Lambda reconcile**: Dejar en blanco (o configurar lambda personalizado)
5. Hacer clic en **Save**

## Configuración de JWT

### 1. Configurar JWT Settings

1. En **Applications** → **Questionnaire Service** → pestaña **JWT**
2. Configurar los siguientes campos:

**JWT Settings:**
- **Enabled**: ✅ Sí
- **Access token signing algorithm**: `RS256` (recomendado) o `HS256`
- **Id token signing algorithm**: `RS256`

**JWT populate lambda** (opcional):
Si necesitas claims personalizados, puedes crear un lambda. Ejemplo básico:

```javascript
function populate(jwt, user, registration) {
  // Agregar roles al JWT
  jwt.roles = registration.roles || [];

  // Agregar email
  jwt.email = user.email;

  // Agregar metadata personalizado (si existe)
  if (user.data) {
    jwt.company_id = user.data.company_id;
    jwt.department = user.data.department;
  }
}
```

**JWT duration:**
- **Access token duration**: `3600` segundos (1 hora)
- **Refresh token duration**: `2592000` segundos (30 días)

3. Hacer clic en **Save**

### 2. Configurar Issuer y Audience

1. Ir a **Tenants** en el menú lateral
2. Seleccionar el tenant usado (ej: `wemoova`)
3. Ir a la pestaña **General**

**Issuer:**
```
https://auth.wemoova.com
```

**Audience** (opcional):
```
questionnaire-service
```

4. Hacer clic en **Save**

## Obtener JWKS Endpoint

El servicio de cuestionarios necesita el endpoint JWKS para validar tokens JWT.

### JWKS Endpoint URL

El endpoint JWKS de FusionAuth sigue este formato:

```
https://auth.wemoova.com/.well-known/jwks.json
```

O específico por aplicación:

```
https://auth.wemoova.com/.well-known/jwks.json?applicationId={APPLICATION_ID}
```

### Verificar JWKS Endpoint

Probar en el navegador o con curl:

```bash
curl https://auth.wemoova.com/.well-known/jwks.json
```

Respuesta esperada:
```json
{
  "keys": [
    {
      "alg": "RS256",
      "kty": "RSA",
      "use": "sig",
      "kid": "abc123...",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

### Configurar en el Servicio

En el archivo `.env` del servicio:

```bash
FUSIONAUTH_URL=https://auth.wemoova.com
```

El middleware JWT construirá automáticamente el endpoint JWKS:
```go
jwksURL := os.Getenv("FUSIONAUTH_URL") + "/.well-known/jwks.json"
```

## Testing de Autenticación

### 1. Obtener Token JWT (Login)

**Endpoint de Login:**
```
POST https://auth.wemoova.com/oauth2/token
```

**Request:**
```bash
curl -X POST https://auth.wemoova.com/oauth2/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password" \
  -d "username=empleado@wemoova.com" \
  -d "password=securePassword123" \
  -d "client_id={YOUR_CLIENT_ID}" \
  -d "client_secret={YOUR_CLIENT_SECRET}"
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImFiYzEyMyJ9...",
  "expires_in": 3600,
  "token_type": "Bearer",
  "refresh_token": "xyz789...",
  "userId": "00000000-0000-0000-0000-000000000001"
}
```

### 2. Decodificar JWT Token

Usar https://jwt.io para inspeccionar el token.

**Payload esperado:**
```json
{
  "sub": "00000000-0000-0000-0000-000000000001",
  "email": "empleado@wemoova.com",
  "roles": ["employee"],
  "iat": 1704067200,
  "exp": 1704070800,
  "iss": "https://auth.wemoova.com",
  "aud": "questionnaire-service"
}
```

### 3. Probar Endpoint del Servicio

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/my-assignments \
  -H "Authorization: Bearer {JWT_TOKEN}"
```

**Response esperada (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "...",
      "questionnaire": {...},
      "status": "pending"
    }
  ]
}
```

**Error de autenticación (401 Unauthorized):**
```json
{
  "error": "Unauthorized",
  "message": "Invalid or expired token"
}
```

## Mapeo de Claims

### Claims Estándar del JWT

| Claim | Descripción | Uso en el Servicio |
|-------|-------------|-------------------|
| `sub` | User ID (UUID de FusionAuth) | Identificador único del usuario, usado como `user_id` en MongoDB |
| `email` | Email del usuario | Información del usuario |
| `roles` | Array de roles asignados | Autorización RBAC |
| `iat` | Issued At (timestamp) | Validación de expiración |
| `exp` | Expiration Time (timestamp) | Validación de expiración |
| `iss` | Issuer (FusionAuth URL) | Validación de origen del token |
| `aud` | Audience (aplicación) | Validación de destino del token |

### Extracción de Claims en el Servicio

El middleware JWT extrae y almacena los claims en el contexto:

```go
type UserClaims struct {
    Sub   string   `json:"sub"`    // User ID
    Email string   `json:"email"`  // Email
    Roles []string `json:"roles"`  // Roles
}

// En el handler:
claims := middleware.GetUserFromContext(r.Context())
userID := claims.Sub
userRoles := claims.Roles
```

## Flujo de Autenticación Completo

```
┌─────────┐          ┌────────────┐          ┌─────────────┐          ┌──────────────┐
│ Usuario │          │  Frontend  │          │ FusionAuth  │          │   Service    │
└────┬────┘          └─────┬──────┘          └──────┬──────┘          └──────┬───────┘
     │                     │                        │                        │
     │  1. Login           │                        │                        │
     ├────────────────────>│                        │                        │
     │                     │  2. POST /oauth2/token │                        │
     │                     ├───────────────────────>│                        │
     │                     │                        │                        │
     │                     │  3. JWT Token          │                        │
     │                     │<───────────────────────┤                        │
     │  4. Token           │                        │                        │
     │<────────────────────┤                        │                        │
     │                     │                        │                        │
     │  5. Request + Token │                        │                        │
     │─────────────────────┼───────────────────────────────────────────────>│
     │                     │                        │                        │
     │                     │                        │  6. Validate JWT       │
     │                     │                        │    (JWKS endpoint)     │
     │                     │                        │<───────────────────────┤
     │                     │                        │                        │
     │                     │                        │  7. JWKS response      │
     │                     │                        ├───────────────────────>│
     │                     │                        │                        │
     │                     │                        │  8. Extract roles      │
     │                     │                        │    Check permissions   │
     │                     │                        │                        │
     │  9. Response        │                        │                        │
     │<─────────────────────────────────────────────────────────────────────┤
     │                     │                        │                        │
```

## Configuración de Usuarios de Prueba

### Crear Usuarios para Testing

**1. Super Admin:**
```
Email: superadmin@wemoova.com
Password: SuperAdmin123!
Roles: super_admin
```

**2. Company Admin:**
```
Email: admin@empresa1.com
Password: Admin123!
Roles: company_admin, employee
```

Después de crear, agregar en MongoDB:
```javascript
db.users_metadata.insertOne({
  "_id": "{FUSION_AUTH_SUB}",
  "company_id": ObjectId("..."),  // ID de empresa en MongoDB
  "supervisor_id": null,
  "department": "Administración",
  "created_at": new Date(),
  "updated_at": new Date()
})
```

**3. Supervisor:**
```
Email: supervisor@empresa1.com
Password: Supervisor123!
Roles: supervisor, employee
```

MongoDB:
```javascript
db.users_metadata.insertOne({
  "_id": "{FUSION_AUTH_SUB}",
  "company_id": ObjectId("..."),
  "supervisor_id": null,  // El supervisor no tiene supervisor
  "department": "IT",
  "created_at": new Date(),
  "updated_at": new Date()
})
```

**4. Employee:**
```
Email: empleado@empresa1.com
Password: Empleado123!
Roles: employee
```

MongoDB:
```javascript
db.users_metadata.insertOne({
  "_id": "{FUSION_AUTH_SUB}",
  "company_id": ObjectId("..."),
  "supervisor_id": "{SUPERVISOR_SUB}",  // ID del supervisor en FusionAuth
  "department": "IT",
  "created_at": new Date(),
  "updated_at": new Date()
})
```

## Troubleshooting

### Error: "Invalid token signature"

**Causa**: El servicio no puede validar la firma del JWT.

**Solución**:
1. Verificar que `FUSIONAUTH_URL` en `.env` sea correcto
2. Probar JWKS endpoint: `curl https://auth.wemoova.com/.well-known/jwks.json`
3. Verificar que el algoritmo de firma sea `RS256` en FusionAuth

### Error: "User does not have required role"

**Causa**: El usuario no tiene el rol requerido para el endpoint.

**Solución**:
1. Verificar roles asignados en FusionAuth Admin → Users → {user} → Registrations
2. Verificar que los nombres de roles coincidan exactamente: `super_admin`, `company_admin`, `supervisor`, `employee`
3. Generar un nuevo token después de actualizar roles

### Error: "User metadata not found"

**Causa**: El usuario no tiene registro en `users_metadata` de MongoDB.

**Solución**:
1. Verificar que existe el registro: `db.users_metadata.findOne({"_id": "{FUSION_AUTH_SUB}"})`
2. Crear metadata usando el endpoint de Super Admin:
   ```bash
   POST /api/v1/users/metadata
   {
     "user_id": "{FUSION_AUTH_SUB}",
     "company_id": "{COMPANY_OBJECT_ID}",
     "department": "IT"
   }
   ```

## Recursos Adicionales

- [FusionAuth Documentation](https://fusionauth.io/docs/)
- [FusionAuth OAuth 2.0 Guide](https://fusionauth.io/docs/v1/tech/oauth/)
- [JWT.io - Token Debugger](https://jwt.io)
- [JWKS Explained](https://auth0.com/docs/secure/tokens/json-web-tokens/json-web-key-sets)

---

**Generado con** [Claude Code](https://claude.com/claude-code)
