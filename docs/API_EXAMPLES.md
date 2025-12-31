# API Examples - Questionnaire Service

Esta guía contiene ejemplos completos de uso de todos los endpoints del servicio de cuestionarios, organizados por rol.

## 📋 Tabla de Contenidos

- [Autenticación](#autenticación)
- [Super Admin Examples](#super-admin-examples)
- [Company Admin Examples](#company-admin-examples)
- [Supervisor Examples](#supervisor-examples)
- [Employee Examples](#employee-examples)
- [Report Examples](#report-examples)

## Autenticación

Todos los endpoints (excepto `/health` y `/ready`) requieren autenticación JWT.

### Obtener Token JWT

```bash
curl -X POST https://auth.wemoova.com/oauth2/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password" \
  -d "username=superadmin@wemoova.com" \
  -d "password=SuperAdmin123!" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "client_secret=YOUR_CLIENT_SECRET"
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "token_type": "Bearer",
  "refresh_token": "xyz789..."
}
```

**Usar el token en requests:**
```bash
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

---

## Super Admin Examples

### 1. Gestión de Cuestionarios

#### Crear Cuestionario

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/questionnaires \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Cuestionario NOM-035 Guía de Referencia III",
    "description": "Identificación y análisis de los factores de riesgo psicosocial y evaluación del entorno organizacional en los centros de trabajo"
  }'
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "Questionnaire created successfully",
  "data": {
    "id": "677e5a2b8f1c2d3e4f5a6b7c",
    "title": "Cuestionario NOM-035 Guía de Referencia III",
    "description": "Identificación y análisis de los factores de riesgo psicosocial...",
    "created_by": "00000000-0000-0000-0000-000000000001",
    "is_active": true,
    "questions": [],
    "created_at": "2025-01-08T10:00:00Z",
    "updated_at": "2025-01-08T10:00:00Z"
  }
}
```

#### Agregar Pregunta - Escala Likert

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/questionnaires/677e5a2b8f1c2d3e4f5a6b7c/questions \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "question_text": "Mi trabajo me permite desarrollar nuevas habilidades",
    "question_type": "likert_scale",
    "options": {
      "min": 1,
      "max": 5,
      "labels": {
        "1": "Nunca",
        "2": "Casi nunca",
        "3": "Algunas veces",
        "4": "Casi siempre",
        "5": "Siempre"
      }
    },
    "is_required": true,
    "order_index": 1
  }'
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "Question added successfully",
  "data": {
    "question_id": "q-uuid-001",
    "question_text": "Mi trabajo me permite desarrollar nuevas habilidades",
    "question_type": "likert_scale",
    "options": {
      "min": 1,
      "max": 5,
      "labels": {
        "1": "Nunca",
        "2": "Casi nunca",
        "3": "Algunas veces",
        "4": "Casi siempre",
        "5": "Siempre"
      }
    },
    "is_required": true,
    "order_index": 1
  }
}
```

#### Agregar Pregunta - Opción Múltiple

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/questionnaires/677e5a2b8f1c2d3e4f5a6b7c/questions \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "question_text": "¿En qué turno trabajas habitualmente?",
    "question_type": "multiple_choice",
    "options": {
      "choices": [
        "Matutino",
        "Vespertino",
        "Nocturno",
        "Mixto/Rotativo"
      ]
    },
    "is_required": true,
    "order_index": 2
  }'
```

#### Agregar Pregunta - Sí/No

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/questionnaires/677e5a2b8f1c2d3e4f5a6b7c/questions \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "question_text": "¿Has recibido capacitación en el último año?",
    "question_type": "yes_no",
    "options": {},
    "is_required": true,
    "order_index": 3
  }'
```

#### Agregar Pregunta - Texto Libre

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/questionnaires/677e5a2b8f1c2d3e4f5a6b7c/questions \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "question_text": "Describe brevemente tu ambiente de trabajo",
    "question_type": "free_text",
    "options": {},
    "is_required": false,
    "order_index": 4
  }'
```

#### Listar Cuestionarios

```bash
curl -X GET "https://qa.services.wemoova.com/questionarie-service/api/v1/questionnaires?page=1&page_size=10" \
  -H "Authorization: Bearer {TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "677e5a2b8f1c2d3e4f5a6b7c",
      "title": "Cuestionario NOM-035 Guía de Referencia III",
      "description": "Identificación y análisis...",
      "created_by": "00000000-0000-0000-0000-000000000001",
      "is_active": true,
      "questions": [
        {
          "question_id": "q-uuid-001",
          "question_text": "Mi trabajo me permite desarrollar nuevas habilidades",
          "question_type": "likert_scale",
          "options": {"min": 1, "max": 5, "labels": {...}},
          "is_required": true,
          "order_index": 1
        }
      ],
      "created_at": "2025-01-08T10:00:00Z"
    }
  ]
}
```

#### Obtener Cuestionario por ID

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/questionnaires/677e5a2b8f1c2d3e4f5a6b7c \
  -H "Authorization: Bearer {TOKEN}"
```

#### Actualizar Cuestionario

```bash
curl -X PUT https://qa.services.wemoova.com/questionarie-service/api/v1/questionnaires/677e5a2b8f1c2d3e4f5a6b7c \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Cuestionario NOM-035 Guía III - Actualizado",
    "description": "Nueva descripción actualizada"
  }'
```

#### Desactivar Cuestionario

```bash
curl -X DELETE https://qa.services.wemoova.com/questionarie-service/api/v1/questionnaires/677e5a2b8f1c2d3e4f5a6b7c \
  -H "Authorization: Bearer {TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Questionnaire deactivated successfully"
}
```

### 2. Gestión de Empresas

#### Crear Empresa

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/companies \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Wemoova Technologies S.A."
  }'
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "Company created successfully",
  "data": {
    "id": "677e5b3c8f1c2d3e4f5a6b7d",
    "name": "Wemoova Technologies S.A.",
    "created_at": "2025-01-08T10:15:00Z",
    "updated_at": "2025-01-08T10:15:00Z"
  }
}
```

#### Listar Empresas

```bash
curl -X GET "https://qa.services.wemoova.com/questionarie-service/api/v1/companies?page=1&page_size=10" \
  -H "Authorization: Bearer {TOKEN}"
```

#### Obtener Empresa por ID

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/companies/677e5b3c8f1c2d3e4f5a6b7d \
  -H "Authorization: Bearer {TOKEN}"
```

#### Actualizar Empresa

```bash
curl -X PUT https://qa.services.wemoova.com/questionarie-service/api/v1/companies/677e5b3c8f1c2d3e4f5a6b7d \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Wemoova Technologies S.A. de C.V."
  }'
```

### 3. Asignar Cuestionario a Empresa

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/companies/677e5b3c8f1c2d3e4f5a6b7d/questionnaires \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "questionnaire_id": "677e5a2b8f1c2d3e4f5a6b7c",
    "period_start": "2025-01-15T00:00:00Z",
    "period_end": "2025-02-15T23:59:59Z"
  }'
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "Questionnaire assigned to company successfully",
  "data": {
    "id": "677e5c4d8f1c2d3e4f5a6b7e",
    "company_id": "677e5b3c8f1c2d3e4f5a6b7d",
    "questionnaire_id": "677e5a2b8f1c2d3e4f5a6b7c",
    "assigned_by": "00000000-0000-0000-0000-000000000001",
    "assigned_at": "2025-01-08T10:30:00Z",
    "period_start": "2025-01-15T00:00:00Z",
    "period_end": "2025-02-15T23:59:59Z",
    "is_active": true
  }
}
```

### 4. Gestión de User Metadata

#### Crear User Metadata

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/users/metadata \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "11111111-1111-1111-1111-111111111111",
    "company_id": "677e5b3c8f1c2d3e4f5a6b7d",
    "supervisor_id": "22222222-2222-2222-2222-222222222222",
    "department": "Tecnología"
  }'
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "User metadata created successfully",
  "data": {
    "user_id": "11111111-1111-1111-1111-111111111111",
    "company_id": "677e5b3c8f1c2d3e4f5a6b7d",
    "supervisor_id": "22222222-2222-2222-2222-222222222222",
    "department": "Tecnología",
    "created_at": "2025-01-08T10:45:00Z",
    "updated_at": "2025-01-08T10:45:00Z"
  }
}
```

#### Obtener User Metadata

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/users/metadata/11111111-1111-1111-1111-111111111111 \
  -H "Authorization: Bearer {TOKEN}"
```

#### Actualizar User Metadata

```bash
curl -X PUT https://qa.services.wemoova.com/questionarie-service/api/v1/users/metadata/11111111-1111-1111-1111-111111111111 \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": "677e5b3c8f1c2d3e4f5a6b7d",
    "supervisor_id": "33333333-3333-3333-3333-333333333333",
    "department": "Innovación y Desarrollo"
  }'
```

#### Listar Usuarios de una Empresa

```bash
curl -X GET "https://qa.services.wemoova.com/questionarie-service/api/v1/companies/677e5b3c8f1c2d3e4f5a6b7d/users?page=1&page_size=20" \
  -H "Authorization: Bearer {TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "user_id": "11111111-1111-1111-1111-111111111111",
      "company_id": "677e5b3c8f1c2d3e4f5a6b7d",
      "supervisor_id": "22222222-2222-2222-2222-222222222222",
      "department": "Tecnología",
      "created_at": "2025-01-08T10:45:00Z"
    },
    {
      "user_id": "44444444-4444-4444-4444-444444444444",
      "company_id": "677e5b3c8f1c2d3e4f5a6b7d",
      "supervisor_id": "22222222-2222-2222-2222-222222222222",
      "department": "Recursos Humanos",
      "created_at": "2025-01-08T11:00:00Z"
    }
  ]
}
```

---

## Company Admin Examples

**Nota**: Company Admin solo puede ver y gestionar cuestionarios de SU empresa.

### 1. Ver Cuestionarios de Mi Empresa

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/companies/677e5b3c8f1c2d3e4f5a6b7d/questionnaires \
  -H "Authorization: Bearer {COMPANY_ADMIN_TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "677e5c4d8f1c2d3e4f5a6b7e",
      "company_id": "677e5b3c8f1c2d3e4f5a6b7d",
      "questionnaire": {
        "id": "677e5a2b8f1c2d3e4f5a6b7c",
        "title": "Cuestionario NOM-035 Guía III",
        "description": "...",
        "questions": [...]
      },
      "period_start": "2025-01-15T00:00:00Z",
      "period_end": "2025-02-15T23:59:59Z",
      "is_active": true
    }
  ]
}
```

### 2. Actualizar Periodo de Cuestionario

```bash
curl -X PUT https://qa.services.wemoova.com/questionarie-service/api/v1/company-questionnaires/677e5c4d8f1c2d3e4f5a6b7e \
  -H "Authorization: Bearer {COMPANY_ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "period_start": "2025-01-20T00:00:00Z",
    "period_end": "2025-03-01T23:59:59Z",
    "is_active": true
  }'
```

### 3. Asignar Cuestionario a Empleados

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/company-questionnaires/677e5c4d8f1c2d3e4f5a6b7e/assignments \
  -H "Authorization: Bearer {COMPANY_ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "user_ids": [
      "11111111-1111-1111-1111-111111111111",
      "44444444-4444-4444-4444-444444444444",
      "55555555-5555-5555-5555-555555555555"
    ]
  }'
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "Assignments created successfully",
  "data": {
    "created_count": 3,
    "assignments": [
      {
        "id": "677e5d5e8f1c2d3e4f5a6b7f",
        "company_questionnaire_id": "677e5c4d8f1c2d3e4f5a6b7e",
        "user_id": "11111111-1111-1111-1111-111111111111",
        "assigned_by": "00000000-0000-0000-0000-000000000002",
        "assigned_at": "2025-01-08T11:30:00Z",
        "status": "pending",
        "responses": []
      },
      {
        "id": "677e5d5e8f1c2d3e4f5a6b80",
        "company_questionnaire_id": "677e5c4d8f1c2d3e4f5a6b7e",
        "user_id": "44444444-4444-4444-4444-444444444444",
        "assigned_by": "00000000-0000-0000-0000-000000000002",
        "assigned_at": "2025-01-08T11:30:00Z",
        "status": "pending",
        "responses": []
      }
    ]
  }
}
```

### 4. Ver Asignaciones de un Cuestionario

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/company-questionnaires/677e5c4d8f1c2d3e4f5a6b7e/assignments \
  -H "Authorization: Bearer {COMPANY_ADMIN_TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "677e5d5e8f1c2d3e4f5a6b7f",
      "company_questionnaire_id": "677e5c4d8f1c2d3e4f5a6b7e",
      "user_id": "11111111-1111-1111-1111-111111111111",
      "status": "in_progress",
      "assigned_at": "2025-01-08T11:30:00Z",
      "started_at": "2025-01-16T09:00:00Z",
      "progress": {
        "answered": 15,
        "total": 50
      }
    },
    {
      "id": "677e5d5e8f1c2d3e4f5a6b80",
      "company_questionnaire_id": "677e5c4d8f1c2d3e4f5a6b7e",
      "user_id": "44444444-4444-4444-4444-444444444444",
      "status": "completed",
      "assigned_at": "2025-01-08T11:30:00Z",
      "started_at": "2025-01-15T14:00:00Z",
      "completed_at": "2025-01-15T15:30:00Z",
      "progress": {
        "answered": 50,
        "total": 50
      }
    }
  ]
}
```

---

## Supervisor Examples

**Nota**: Supervisor solo puede asignar cuestionarios a SU equipo (empleados con `supervisor_id` = su user_id).

### 1. Ver Cuestionarios de Mi Empresa

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/my-company/questionnaires \
  -H "Authorization: Bearer {SUPERVISOR_TOKEN}"
```

### 2. Asignar Cuestionario a Mi Equipo

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/company-questionnaires/677e5c4d8f1c2d3e4f5a6b7e/assignments \
  -H "Authorization: Bearer {SUPERVISOR_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "user_ids": [
      "11111111-1111-1111-1111-111111111111",
      "66666666-6666-6666-6666-666666666666"
    ]
  }'
```

**Nota**: Solo puede asignar a empleados donde `supervisor_id` = su user_id. Si intenta asignar a empleados de otro supervisor, recibirá error 403 Forbidden.

### 3. Ver Asignaciones de Mi Equipo

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/my-team/assignments \
  -H "Authorization: Bearer {SUPERVISOR_TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "677e5d5e8f1c2d3e4f5a6b7f",
      "user_id": "11111111-1111-1111-1111-111111111111",
      "questionnaire_title": "Cuestionario NOM-035 Guía III",
      "status": "in_progress",
      "assigned_at": "2025-01-08T11:30:00Z",
      "progress": {
        "answered": 15,
        "total": 50
      }
    }
  ]
}
```

---

## Employee Examples

**Nota**: Empleado solo puede ver y responder cuestionarios asignados a SÍ MISMO.

### 1. Ver Mis Asignaciones

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/my-assignments \
  -H "Authorization: Bearer {EMPLOYEE_TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "677e5d5e8f1c2d3e4f5a6b7f",
      "questionnaire": {
        "id": "677e5a2b8f1c2d3e4f5a6b7c",
        "title": "Cuestionario NOM-035 Guía III",
        "description": "Identificación y análisis..."
      },
      "status": "pending",
      "period": {
        "start": "2025-01-15T00:00:00Z",
        "end": "2025-02-15T23:59:59Z"
      },
      "assigned_at": "2025-01-08T11:30:00Z",
      "progress": {
        "answered": 0,
        "total": 50
      }
    },
    {
      "id": "677e5d5e8f1c2d3e4f5a6b81",
      "questionnaire": {
        "id": "677e5a2b8f1c2d3e4f5a6b9c",
        "title": "Evaluación de Clima Laboral",
        "description": "..."
      },
      "status": "in_progress",
      "period": {
        "start": "2025-01-10T00:00:00Z",
        "end": "2025-01-31T23:59:59Z"
      },
      "assigned_at": "2025-01-05T09:00:00Z",
      "started_at": "2025-01-12T14:30:00Z",
      "progress": {
        "answered": 12,
        "total": 30
      }
    }
  ]
}
```

### 2. Ver Detalle de Asignación (con preguntas)

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/assignments/677e5d5e8f1c2d3e4f5a6b7f \
  -H "Authorization: Bearer {EMPLOYEE_TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "677e5d5e8f1c2d3e4f5a6b7f",
    "company_questionnaire_id": "677e5c4d8f1c2d3e4f5a6b7e",
    "user_id": "11111111-1111-1111-1111-111111111111",
    "status": "pending",
    "assigned_at": "2025-01-08T11:30:00Z",
    "questionnaire": {
      "id": "677e5a2b8f1c2d3e4f5a6b7c",
      "title": "Cuestionario NOM-035 Guía III",
      "description": "...",
      "questions": [
        {
          "question_id": "q-uuid-001",
          "question_text": "Mi trabajo me permite desarrollar nuevas habilidades",
          "question_type": "likert_scale",
          "options": {
            "min": 1,
            "max": 5,
            "labels": {
              "1": "Nunca",
              "2": "Casi nunca",
              "3": "Algunas veces",
              "4": "Casi siempre",
              "5": "Siempre"
            }
          },
          "is_required": true,
          "order_index": 1
        },
        {
          "question_id": "q-uuid-002",
          "question_text": "¿En qué turno trabajas habitualmente?",
          "question_type": "multiple_choice",
          "options": {
            "choices": ["Matutino", "Vespertino", "Nocturno", "Mixto/Rotativo"]
          },
          "is_required": true,
          "order_index": 2
        }
      ]
    },
    "responses": [],
    "progress": {
      "answered": 0,
      "total": 50
    }
  }
}
```

### 3. Guardar Respuesta Individual

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/assignments/677e5d5e8f1c2d3e4f5a6b7f/responses \
  -H "Authorization: Bearer {EMPLOYEE_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "question_id": "q-uuid-001",
    "response_value": 4
  }'
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Response saved successfully",
  "data": {
    "assignment_id": "677e5d5e8f1c2d3e4f5a6b7f",
    "question_id": "q-uuid-001",
    "response_value": 4,
    "answered_at": "2025-01-16T09:15:00Z"
  }
}
```

### 4. Actualizar Múltiples Respuestas (Guardado Masivo)

```bash
curl -X PUT https://qa.services.wemoova.com/questionarie-service/api/v1/assignments/677e5d5e8f1c2d3e4f5a6b7f/responses \
  -H "Authorization: Bearer {EMPLOYEE_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "responses": [
      {
        "question_id": "q-uuid-001",
        "response_value": 5
      },
      {
        "question_id": "q-uuid-002",
        "response_value": "Matutino"
      },
      {
        "question_id": "q-uuid-003",
        "response_value": true
      },
      {
        "question_id": "q-uuid-004",
        "response_value": "El ambiente es colaborativo y respetuoso"
      }
    ]
  }'
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Responses updated successfully",
  "data": {
    "updated_count": 4,
    "progress": {
      "answered": 4,
      "total": 50
    }
  }
}
```

### 5. Enviar Cuestionario Completado

```bash
curl -X POST https://qa.services.wemoova.com/questionarie-service/api/v1/assignments/677e5d5e8f1c2d3e4f5a6b7f/submit \
  -H "Authorization: Bearer {EMPLOYEE_TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Questionnaire submitted successfully",
  "data": {
    "assignment_id": "677e5d5e8f1c2d3e4f5a6b7f",
    "status": "completed",
    "completed_at": "2025-01-16T10:30:00Z",
    "total_responses": 50
  }
}
```

**Error - Preguntas Requeridas Faltantes (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Cannot submit: 5 required questions not answered",
  "details": {
    "missing_questions": [
      "q-uuid-010",
      "q-uuid-015",
      "q-uuid-020",
      "q-uuid-025",
      "q-uuid-030"
    ]
  }
}
```

---

## Report Examples

### 1. Métricas de Completitud de Cuestionario

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/reports/company-questionnaire/677e5c4d8f1c2d3e4f5a6b7e/completion \
  -H "Authorization: Bearer {COMPANY_ADMIN_TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "company_questionnaire_id": "677e5c4d8f1c2d3e4f5a6b7e",
    "questionnaire_title": "Cuestionario NOM-035 Guía III",
    "company_name": "Wemoova Technologies S.A.",
    "period": {
      "start": "2025-01-15T00:00:00Z",
      "end": "2025-02-15T23:59:59Z"
    },
    "completion_metrics": {
      "total_employees": 120,
      "total_assigned": 100,
      "not_started": 15,
      "in_progress": 30,
      "completed": 55,
      "completion_percentage": 55.0,
      "average_time_to_complete_minutes": 48.5
    },
    "completion_by_department": [
      {
        "department": "Tecnología",
        "total": 30,
        "completed": 25,
        "percentage": 83.33
      },
      {
        "department": "Recursos Humanos",
        "total": 15,
        "completed": 15,
        "percentage": 100.0
      },
      {
        "department": "Operaciones",
        "total": 25,
        "completed": 10,
        "percentage": 40.0
      },
      {
        "department": "Ventas",
        "total": 30,
        "completed": 5,
        "percentage": 16.67
      }
    ]
  }
}
```

### 2. Overview de Empresa (Todos los Cuestionarios)

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/reports/company/677e5b3c8f1c2d3e4f5a6b7d/overview \
  -H "Authorization: Bearer {COMPANY_ADMIN_TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "company_id": "677e5b3c8f1c2d3e4f5a6b7d",
    "company_name": "Wemoova Technologies S.A.",
    "total_employees": 120,
    "questionnaires": [
      {
        "company_questionnaire_id": "677e5c4d8f1c2d3e4f5a6b7e",
        "questionnaire_id": "677e5a2b8f1c2d3e4f5a6b7c",
        "title": "Cuestionario NOM-035 Guía III",
        "period": {
          "start": "2025-01-15T00:00:00Z",
          "end": "2025-02-15T23:59:59Z"
        },
        "is_active": true,
        "metrics": {
          "total_assigned": 100,
          "completed": 55,
          "completion_percentage": 55.0
        }
      },
      {
        "company_questionnaire_id": "677e5c4d8f1c2d3e4f5a6b9f",
        "questionnaire_id": "677e5a2b8f1c2d3e4f5a6b9c",
        "title": "Evaluación de Clima Laboral",
        "period": {
          "start": "2025-01-10T00:00:00Z",
          "end": "2025-01-31T23:59:59Z"
        },
        "is_active": true,
        "metrics": {
          "total_assigned": 80,
          "completed": 72,
          "completion_percentage": 90.0
        }
      }
    ],
    "overall_metrics": {
      "total_questionnaires": 2,
      "total_assignments": 180,
      "total_completed": 127,
      "overall_completion_percentage": 70.56
    }
  }
}
```

### 3. Progreso de Empleados

```bash
curl -X GET https://qa.services.wemoova.com/questionarie-service/api/v1/reports/company/677e5b3c8f1c2d3e4f5a6b7d/employees-progress \
  -H "Authorization: Bearer {COMPANY_ADMIN_TOKEN}"
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "company_id": "677e5b3c8f1c2d3e4f5a6b7d",
    "company_name": "Wemoova Technologies S.A.",
    "employees": [
      {
        "user_id": "11111111-1111-1111-1111-111111111111",
        "department": "Tecnología",
        "supervisor_id": "22222222-2222-2222-2222-222222222222",
        "assignments": [
          {
            "questionnaire_title": "Cuestionario NOM-035 Guía III",
            "status": "completed",
            "completion_percentage": 100.0,
            "completed_at": "2025-01-20T16:45:00Z"
          },
          {
            "questionnaire_title": "Evaluación de Clima Laboral",
            "status": "in_progress",
            "completion_percentage": 60.0
          }
        ],
        "overall_completion": 80.0
      },
      {
        "user_id": "44444444-4444-4444-4444-444444444444",
        "department": "Recursos Humanos",
        "supervisor_id": "22222222-2222-2222-2222-222222222222",
        "assignments": [
          {
            "questionnaire_title": "Cuestionario NOM-035 Guía III",
            "status": "pending",
            "completion_percentage": 0.0
          }
        ],
        "overall_completion": 0.0
      }
    ]
  }
}
```

**Nota**: Los reportes NO incluyen respuestas individuales de empleados, solo datos agregados para proteger la privacidad.

---

## Códigos de Error Comunes

| Código | Descripción | Ejemplo |
|--------|-------------|---------|
| 400 | Bad Request - Parámetros inválidos | `{"error": "Bad Request", "message": "company_id is required"}` |
| 401 | Unauthorized - Token JWT inválido o expirado | `{"error": "Unauthorized", "message": "Invalid or expired token"}` |
| 403 | Forbidden - Usuario no tiene permisos | `{"error": "Forbidden", "message": "User does not have required role"}` |
| 404 | Not Found - Recurso no encontrado | `{"error": "Not Found", "message": "Questionnaire not found"}` |
| 409 | Conflict - Conflicto de datos | `{"error": "Conflict", "message": "User already assigned to this questionnaire"}` |
| 500 | Internal Server Error - Error del servidor | `{"error": "Internal Server Error", "message": "Database connection failed"}` |

---

**Generado con** [Claude Code](https://claude.com/claude-code)
