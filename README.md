# FeedFlow

FeedFlow is a small full-stack social media application with a Go REST API, React frontend, JWT authentication, Elasticsearch search, local or Google Cloud Storage media uploads, and OpenAI-powered post text generation.

## Architecture

- `main.go` starts a Go HTTP server on port `8080`.
- `handler/` exposes REST endpoints with Gorilla Mux and JWT middleware.
- `service/` contains authentication, post, search, and OpenAI generation logic.
- `backend/` wraps Elasticsearch and media storage clients. Media can be stored locally for development or in Google Cloud Storage for deployment.
- `frontend/` contains the React/Vite client.

Data flow:

```text
React client -> Go API -> Elasticsearch
React client -> Go API -> Local storage or Google Cloud Storage
React client -> Go API -> OpenAI Responses API
```

## Features

- Sign up and sign in with JWT authentication.
- Hash passwords with bcrypt before storing users.
- Upload image or video posts to local disk in development or Google Cloud Storage in production.
- Store and search user and post metadata in Elasticsearch.
- Generate social post text from a prompt using the OpenAI Responses API.
- Browse posts with keyword/user filters and pagination.
- Delete posts owned by the authenticated user.

## API

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/signup` | No | Create a user account. |
| `POST` | `/signin` | No | Validate credentials and return a JWT. |
| `POST` | `/ai/post` | Yes | Generate post text from a prompt. |
| `POST` | `/upload` | Yes | Upload media and create a post. |
| `GET` | `/search` | Yes | Search posts by `keywords`, `user`, `page`, and `pageSize`. |
| `DELETE` | `/post/{id}` | Yes | Delete a post owned by the current user. |

## Environment

Copy `.env.example` and set real values:

```text
JWT_SECRET=replace-with-a-long-random-secret
ES_URL=http://localhost:9200
ES_USERNAME=
ES_PASSWORD=
GCS_BUCKET=your-gcs-bucket
STORAGE_BACKEND=local
LOCAL_STORAGE_DIR=uploads
LOCAL_MEDIA_BASE_URL=http://localhost:8080/media
OPENAI_API_KEY=sk-your-openai-api-key
OPENAI_MODEL=gpt-5
```

Use `STORAGE_BACKEND=local` for local development and load testing without Google Cloud credentials. Uploaded media is written to `LOCAL_STORAGE_DIR` and served at `/media/{object_id}`.

To use Google Cloud Storage instead, remove `STORAGE_BACKEND=local`, set `GCS_BUCKET`, and run the backend with Google Cloud credentials available in the environment.

For the frontend, copy `frontend/.env.example` if the API is not running at `http://localhost:8080`.

## Run Locally

Backend:

```bash
go run .
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

## Deployment

`app.yaml` configures the Go backend for Google App Engine flexible environment with automatic scaling between one and two instances. Deploy with:

```bash
gcloud app deploy
```
