# questionarie-service

Servicio de administracion de cuestionarios de WeMoova

## Overview

This microservice is part of the WeMoova platform and is deployed via GitOps using Argo CD.

**Technology Stack:**
- **Language**: Go 1.21+
- **Router**: Chi
- **Database**: PostgreSQL (shared RDS with schema isolation)
- **Authentication**: FusionAuth JWT

## API Endpoints

### Public Endpoints

- `GET /health` - Basic health check
- `GET /ready` - Readiness probe (checks database connection)
- `GET /live` - Liveness probe

### Protected Endpoints

All other endpoints require a valid JWT token from FusionAuth.

**Example:**
```bash
curl -H "Authorization: Bearer <token>" \
  https://services.wemoova.com/questionarie-service/api/v1/example
```

## Local Development

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 15+
- FusionAuth instance

### Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/WeMoova/questionarie-service.git
   cd questionarie-service
   ```

2. **Configure environment variables:**
   ```bash
   cp .env.example .env
   # Edit .env with your local configuration
   ```

3. **Install dependencies:**
   ```bash
   go mod download
   ```

4. **Run database migrations:**
   ```bash
   goose -dir migrations postgres "postgresql://user:password@localhost:5432/dbname?search_path=questionarie_service" up
   ```

5. **Start the development server:**
   ```bash
   go run main.go
   ```

The server will start on `http://localhost:8080`

## Database

This service uses a **shared PostgreSQL RDS instance** with **schema isolation**. Each service has its own schema:

- Schema name: `questionarie_service`
- All queries automatically use `search_path` to target the correct schema

### Migrations

**Create a new migration:**
```bash
goose -dir migrations create <migration_name> sql
```

**Run migrations:**
```bash
goose -dir migrations postgres "postgresql://..." up
```

**Rollback:**
```bash
goose -dir migrations postgres "postgresql://..." down
```

## Authentication

This service uses **FusionAuth** for authentication. All protected endpoints require a valid JWT token.

### Getting a Token

1. Login via FusionAuth:
   ```bash
   curl -X POST https://auth.wemoova.com/api/login \
     -H "Content-Type: application/json" \
     -d '{"loginId":"user@example.com","password":"password"}'
   ```

2. Extract the `token` from the response

3. Use the token in API requests:
   ```bash
   curl -H "Authorization: Bearer <token>" \
     https://services.wemoova.com/questionarie-service/api/v1/example
   ```

## Deployment

This service uses **GitHub Actions** for CI/CD and **Argo CD** for GitOps deployment.

### Environments

- **QA**: `https://qa.services.wemoova.com/questionarie-service/`
- **Production**: `https://services.wemoova.com/questionarie-service/`

### Deploying to QA

```bash
gh release create v1.0.0-qa --generate-notes
```

### Deploying to Production

```bash
gh release create v1.0.0 --generate-notes
```

### Deployment Flow

1. Create GitHub release (with tag `vX.X.X` or `vX.X.X-qa`)
2. GitHub Actions builds Docker image
3. Image pushed to `ghcr.io/WeMoova/questionarie-service:<tag>`
4. GitHub Actions updates Kustomize manifests in `argo-apps` repo
5. Argo CD detects change and deploys to Kubernetes

## Monitoring

### Health Checks

- **Health**: `GET /health` - Basic service health
- **Ready**: `GET /ready` - Service + database readiness
- **Live**: `GET /live` - Service liveness

### Logs

**View logs in Kubernetes:**
```bash
# Production
kubectl logs -f deployment/questionarie-service -n questionarie-service

# QA
kubectl logs -f deployment/questionarie-service-qa -n questionarie-service-qa
```

## Architecture

### Path-Based Routing

This service is accessible via path-based routing on a shared ALB:

- Production: `https://services.wemoova.com/questionarie-service/*`
- QA: `https://qa.services.wemoova.com/questionarie-service/*`

The ALB automatically strips the `/questionarie-service` prefix before forwarding to the service, so your application sees:
- `https://services.wemoova.com/questionarie-service/api/v1/users` → `/api/v1/users`

### Kubernetes Resources

- **Namespace**: `questionarie-service` (production) or `questionarie-service-qa` (QA)
- **Deployment**: Manages pod replicas
- **Service**: ClusterIP on port 8080
- **Ingress**: ALB with path-based routing
- **ConfigMap**: Non-sensitive configuration
- **Secret**: Database credentials

## Development Guidelines

### Code Style

- Follow standard Go formatting (`gofmt`)
- Use `golangci-lint` for linting
- Write tests for all handlers

### Adding New Endpoints

1. Create handler/route file
2. Register route in main application
3. Add authentication middleware if needed
4. Write tests
5. Update this README

## Troubleshooting

### Database Connection Issues

- Verify `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` in `.env`
- Check if PostgreSQL is running and accessible
- Verify schema exists: `SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'questionarie_service';`

### Authentication Issues

- Verify `FUSIONAUTH_URL` is correct
- Check token expiration
- Ensure JWKS endpoint is accessible: `curl https://auth.wemoova.com/.well-known/jwks.json`

### Deployment Issues

- Check GitHub Actions workflow logs
- Check Argo CD application status: `kubectl get applications -n argocd`
- Verify Kustomize manifests: `kubectl kustomize manifests/overlays/production`

## Support

For issues or questions:
- GitHub Issues: https://github.com/WeMoova/questionarie-service/issues
- WeMoova Team: team@wemoova.com

## License

MIT License - WeMoova © 2024
