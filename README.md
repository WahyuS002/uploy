# Uploy

A lightweight self-hosted application deployment platform. Uploy allows you to manage projects, services, environments, servers, SSH keys, and custom domains with automated deployment reconciliations.

## Features

- Project & Service Management: Organize applications and databases into projects.
- Multi-Environment Support: Configure Development, Staging, and Production environments.
- Server & SSH Access: Manage remote servers via SSH keys for deployments.
- Domain & Ingress Management: Automatic domain reconciliation and SSL/ingress handling.
- Authentication & Security: Cookie-based sessions, GitHub/Google OAuth2, and AES-256 encrypted secrets at rest.
- Modern Dashboard: Built with Svelte 5 runes, Tailwind CSS v4, and responsive inspector side panels.

## Tech Stack

### Backend (`/api`)
- Go 1.25+
- `net/http` standard library with `oapi-codegen` OpenAPI 3 router
- PostgreSQL 18 with `pgx/v5` and `sqlc`
- Database migrations with `goose`
- Hot reload with `air`

### Frontend (`/frontend`)
- SvelteKit (Svelte 5)
- Tailwind CSS v4, Bits UI, Lucide Icons
- TypeScript, Vite, `pnpm`

---

## Quickstart

### 1. Start Database

Start PostgreSQL via Docker Compose:

```bash
docker compose up -d
```

### 2. Backend Setup (`/api`)

```bash
cd api
cp .env.example .env
# Edit .env and set ENCRYPTION_KEY (e.g. openssl rand -hex 32)

# Run database migrations
go run ./cmd/migrate

# Start development server
air # or go run .
```

The API server runs at `http://localhost:8080`.

### 3. Frontend Setup (`/frontend`)

```bash
cd frontend
pnpm install
pnpm dev
```

Open `http://localhost:5173` in your browser.

---

## Common Scripts

### Backend (`/api`)
- `make dev`: Run dev server with Air
- `make migrate-up`: Run database migrations
- `make generate`: Run `sqlc` and `oapi-codegen`

### Frontend (`/frontend`)
- `pnpm dev`: Start Vite development server
- `pnpm build`: Build production bundle
- `pnpm lint`: Run Prettier and ESLint checks
- `pnpm format`: Format frontend code
- `pnpm generate:api`: Generate TypeScript types from OpenAPI spec

## License

Uploy is licensed under the [Apache License 2.0](LICENSE).
