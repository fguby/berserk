# Manga AI Backend

Go + Echo API for the Manga AI web image studio. The older iOS app routes are still present for compatibility; the web image generation route uses the XAI relay-compatible Responses API configured by `XAI_*` settings.

## Run locally

```bash
docker compose up -d postgres
go mod tidy
XAI_API_KEY=your_xai_relay_key go run ./cmd/api
```

The API listens on `http://127.0.0.1:8080` by default.

Configuration is loaded from `config.yaml` by default. Set `CONFIG_PATH=/path/to/config.yaml` to use another file; environment variables such as `DATABASE_URL` and `PUBLIC_BASE_URL` still override the YAML values for deployment.
The API is exposed under `/berserk/api/v1/...`; generated local images are served from `/berserk/generated/...`.

## XAI relay settings

- `XAI_API_KEY`: relay API key. Keep this in the environment instead of committing it.
- `XAI_BASE_URL`: relay base URL, defaults to `https://api-xai.ainaibahub.com/v1`.
- `XAI_RESPONSES_PATH`: Responses path, defaults to `/responses`.
- `XAI_MAIN_MODEL`: orchestration model, defaults to `gpt-5.5`.
- `XAI_IMAGE_MODEL`: image tool model, defaults to `gpt-image-2`.

## Image storage settings

New generated images are uploaded to Cloudflare R2 when all `R2_*` settings are present. Existing `oss://...` records continue to be readable through the old OSS signing path.

- `R2_BUCKET`: R2 bucket name.
- `R2_ENDPOINT`: S3-compatible endpoint, usually `https://<account_id>.r2.cloudflarestorage.com`.
- `R2_ACCESS_KEY_ID`, `R2_ACCESS_KEY_SECRET`: R2 API token credentials.
- `R2_OBJECT_PREFIX`: object prefix, defaults to `berserk/generated`.
- `R2_SIGNED_URL_TTL_SECONDS`: signed URL lifetime for gallery and detail image display, defaults to `3600`.

## Aliyun LLM settings

Studio story analysis uses Alibaba Cloud DashScope OpenAI-compatible chat completions.

- `ALIYUN_LLM_API_KEY` or `DASHSCOPE_API_KEY`: DashScope API key.
- `ALIYUN_LLM_BASE_URL`: defaults to `https://dashscope.aliyuncs.com/compatible-mode/v1`.
- `ALIYUN_LLM_MODEL`: defaults to `qwen3.6-plus`.

## Email auth settings

- `WEB_APP_ID`: app id used for the web site account namespace, defaults to `mangaai.web`.
- `EMAIL_CODE_TTL_SECONDS`: email verification code validity window, defaults to `90`.
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`: SMTP connection settings.
- `SMTP_FROM_EMAIL`, `SMTP_FROM_NAME`: sender address and display name.
- `SMTP_TLS_MODE`: `starttls`, `tls`, or `none`; defaults to `starttls`.

## Endpoints

- `GET /berserk/healthz`
- `GET /berserk/api/v1/images/gallery`
- `POST /berserk/api/v1/images/generate`
- `POST /berserk/api/v1/web/images/generate`
- `GET /berserk/api/v1/credits/packages`
- `POST /berserk/api/v1/credits/purchase`
- `POST /berserk/api/v1/auth/email/code`
- `POST /berserk/api/v1/auth/email/verify`
- `POST /berserk/api/v1/auth/email/register`
- `POST /berserk/api/v1/auth/email/login`
- `POST /berserk/api/v1/auth/email/reset`
- `POST /berserk/api/v1/auth/apple`
- `GET /berserk/api/v1/me`
- `POST /berserk/api/v1/subscriptions`
- `POST /berserk/api/v1/subscriptions/restore`
- `GET /berserk/api/v1/templates`
- `GET /berserk/api/v1/copywriting?category=travel`
- `GET /berserk/api/v1/tickets`
- `POST /berserk/api/v1/tickets`
- `GET /berserk/api/v1/tickets/:id`
- `DELETE /berserk/api/v1/tickets/:id`
- `POST /berserk/api/v1/memories`
- `GET /berserk/api/v1/memories/:id`
- `POST /berserk/api/v1/feedback`
- `POST /berserk/api/v1/manga/generate`
- `POST /berserk/api/v1/manga/script`
- `POST /berserk/api/v1/studio/story/analyze`
- `GET /berserk/api/v1/studio/projects`
- `POST /berserk/api/v1/studio/projects`
- `GET /berserk/api/v1/studio/projects/:id`
- `GET /berserk/api/v1/studio/assets`
- `PATCH /berserk/api/v1/studio/assets/:id`
- `POST /berserk/api/v1/studio/assets/:id/generate`
- `GET /m/:id`

### Web image generation

`POST /berserk/api/v1/images/generate` requires `Authorization: Bearer <token>` and consumes 5 credits for each request. Generated images are saved to the web gallery with their prompt, style, image data, size, quality, and credit cost.

Request:

```json
{
  "prompt": "雨夜街头的赛博朋克少女，霓虹招牌反射在积水里",
  "style": "赛博朋克",
  "images": ["data:image/jpeg;base64,..."],
  "n": 1,
  "size": "1024x1536",
  "quality": "medium"
}
```

Response images are returned as browser-ready data URLs:

```json
{
  "images": [
    {
      "url": "data:image/png;base64,...",
      "mimeType": "image/png"
    }
  ],
  "prompt": "雨夜街头的赛博朋克少女，霓虹招牌反射在积水里",
  "style": "赛博朋克",
  "size": "1024x1536",
  "quality": "medium",
  "credits": 5,
  "user": {
    "credits": 95
  },
  "createdAt": "2026-05-08T00:00:00Z"
}
```

Load the waterfall gallery:

```http
GET /berserk/api/v1/images/gallery?limit=30
```

The response is `{ "items": [...] }`; each item contains `id`, `image`, `prompt`, `style`, `tag`, `ratio`, `size`, `quality`, and `createdAt`.

### AI Manga Studio

`GET /berserk/api/v1/studio/projects` returns the user's works. `POST /berserk/api/v1/studio/projects` creates a new work, and `GET /berserk/api/v1/studio/projects/:id` returns the work plus its saved assets.

`POST /berserk/api/v1/studio/story/analyze` requires `Authorization: Bearer <token>`. It sends the story to the configured Aliyun text model, saves one analysis row, and creates draft assets for characters, scenes, props, and dialogue rules. Pass `analysisID` to update an existing work.

```json
{
  "analysisID": "existing-project-id",
  "projectName": "雨夜猎妖少女",
  "story": "雨夜，少女走进废弃车站……",
  "style": "黑白漫画 / 赛博朋克"
}
```

`GET /berserk/api/v1/studio/assets?type=character` returns saved assets. `PATCH /berserk/api/v1/studio/assets/:id` updates editable fields such as `prompt`. `POST /berserk/api/v1/studio/assets/:id/generate` consumes 5 credits, generates a reference image through the existing image service, and writes the image back to the asset library.

### Credits

Credit packages:

- `credits_100`: 100 credits, 10 RMB
- `credits_500`: 500 credits, 49 RMB
- `credits_1000`: 1000 credits, 95 RMB
- `credits_5000`: 5000 credits, 450 RMB

`GET /berserk/api/v1/credits/packages` returns these packages. `POST /berserk/api/v1/credits/purchase` requires `Authorization: Bearer <token>` and currently treats payment as successful by default, creates a paid mock order, writes a credit ledger entry, and returns the refreshed user balance.

The database design uses:

- `user_credit_accounts`: current balance and lifetime recharge/spend totals.
- `credit_orders`: package purchase order records, ready to attach a future payment provider order id.
- `credit_ledger`: immutable credit changes, including purchases, generation spends, and refunds.

### Email registration and login

Request a code:

```json
{
  "email": "name@example.com",
  "mode": "register"
}
```

The endpoint accepts `"mode": "register"` for registration and `"mode": "reset"` for password reset. A successful response returns `expiresIn: 90` by default and `expiresAt` in UTC.

Verify the received code before showing the password setup form:

```json
{
  "email": "name@example.com",
  "mode": "register",
  "code": "123456"
}
```

The response includes a short-lived `setupToken`. Register or reset with that token and the new password:

```json
{
  "email": "name@example.com",
  "setupToken": "returned-by-/auth/email/verify",
  "password": "your-password"
}
```

Log in with password:

```json
{
  "email": "name@example.com",
  "password": "your-password"
}
```

`POST /berserk/api/v1/auth/email/reset` consumes a verified reset setup token, updates the password, and returns a fresh session. All email auth responses share the Apple auth shape: `{ "token": "...", "user": { ... } }`.
