# Berserk AI

Berserk AI is a standalone image studio service with:

- `studio`: React/Vite frontend.
- `backend`: Go API service for auth, credits, gallery, image generation, OSS signed images, favorites, and seed data.

## Local Development

Backend:

```bash
cd backend
cp config.example.yaml config.yaml
go run ./cmd/api
```

Frontend:

```bash
cd studio
npm install
npm run dev
```

The frontend points to `http://127.0.0.1:8080/pk` by default.

## Verification

```bash
cd backend && go test ./...
cd studio && npm run build
```
