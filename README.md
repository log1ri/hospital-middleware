# Hospital Middleware

A middleware service that lets hospital staff register, sign in, and look up
patients. Patient records are fetched from Hospital A's HIS and cached locally,
so a staff member only ever sees patients belonging to their own hospital.

**Stack:** Go 1.25 · Gin · PostgreSQL 18 · nginx · Docker Compose

---

## Running it

```bash
git clone <repo-url> && cd hospital-middleware
cp .env.example .env          # fill in POSTGRES_PASSWORD and JWT_SECRET
docker compose up -d --build
```

The API is then at `http://localhost:8080`. Generate the two secrets with
`openssl rand -hex 32`; they have no defaults on purpose, so an unconfigured
deployment fails at startup instead of running with a guessable signing key.

Four containers come up: **nginx** (the only one published to the host),
**app**, **postgres**, and **his-a-dummy**. The last one is a small stand-in for
Hospital A's HIS, described at the end of this document.

Run the tests with `go test ./...`.

---

## 1. Project Structure

```
cmd/
  api/                  service entry point: loads config, wires dependencies, starts the server
  his-a-dummy/          stand-in for Hospital A's HIS (development only)
internal/
  routes/               URL → handler table, the one place every endpoint is registered
  handlers/             HTTP concerns only: bind the request, map errors to status codes
  middlewares/          JWT authentication; puts staff_id and hospital into the request context
  services/             business rules: cache-then-HIS lookup, credential checks, filtering
  repositories/         SQL against PostgreSQL through GORM
  clients/              outbound HTTP to Hospital A's HIS
  models/               database models and request/response types
  config/               environment loading and validation
  db/                   connection and AutoMigrate
  utils/                password hashing and JWT signing
nginx/                  reverse proxy config template
```

### How a request travels

```
client → nginx → routes → middleware → handler → service → repository → PostgreSQL
                                                     └────→ client ──→ Hospital A HIS
```

Each layer has one job and depends only on the layer beneath it:

- **handlers** never touch the database and never decide business rules. They
  bind input, call one service method, and translate its error into a status
  code.
- **services** hold every rule that matters — that a search checks the local
  cache before reaching out to the HIS, that a staff member is scoped to their
  own hospital, that an HIS response must actually describe the patient we asked
  for. They depend on interfaces, not on concrete repositories, which is what
  lets the tests substitute in-memory stores.
- **repositories** and **clients** are the only places that know about SQL and
  HTTP respectively.

### Testing

`go test ./...` covers **84.1%** of statements. All three APIs are tested for
both success and failure cases:

| Area | What is covered |
|---|---|
| `/staff/create` | success, duplicate username, weak password, non-alphanumeric username, missing field, malformed JSON |
| `/staff/login` | token issued and parseable, wrong password, wrong hospital, unknown user, database failure |
| `/patient/search` | cache hit, HIS hit, every filter, missing token, HIS returning the wrong patient, HIS down, database down |
| Filtering logic | Thai and English name matching, whole-word matching, absent identifiers |
| Config | required variables, invalid port, invalid environment, malformed `.env` |

Everything above runs without any external dependency. The repository tests are
integration tests against a real PostgreSQL instance, so bring the stack up
first — `docker compose up -d postgres` is enough — before running the suite.

---

## 2. API Spec

Base URL: `http://localhost:8080`. All request and response bodies are JSON.

### `GET /health`

Returns `200` with `{"status":"ok"}`. Used by the container health check.

---

### `POST /staff/create`

Registers a staff member.

```json
{ "username": "nurse01", "password": "secret123", "hospital": "A" }
```

| Field | Rules |
|---|---|
| `username` | required, alphanumeric, unique |
| `password` | required, at least 6 characters |
| `hospital` | required |

Passwords are hashed with bcrypt before storage; the plaintext is never
persisted or logged.

| Status | Body |
|---|---|
| `201` | `{"message":"staff created"}` |
| `400` | `{"error":"invalid request body"}` |
| `409` | `{"error":"username already exists"}` |
| `500` | `{"error":"could not create staff"}` |

```bash
curl -X POST http://localhost:8080/staff/create \
  -H 'Content-Type: application/json' \
  -d '{"username":"nurse01","password":"secret123","hospital":"A"}'
```

---

### `POST /staff/login`

Exchanges credentials for a JWT. The hospital is part of the credential: a
staff member cannot sign in against a hospital they do not belong to, even with
the correct password.

```json
{ "username": "nurse01", "password": "secret123", "hospital": "A" }
```

| Status | Body |
|---|---|
| `200` | `{"token":"eyJhbGciOiJIUzI1NiIs..."}` |
| `400` | `{"error":"invalid request body"}` |
| `401` | `{"error":"invalid credentials"}` |
| `500` | `{"error":"could not log in"}` |

A wrong password, an unknown username, and a hospital mismatch all return the
same `401`, so the response cannot be used to discover which usernames exist.

The token is HS256 and carries `staff_id`, `hospital`, `iat` and `exp`. Its
lifetime comes from `JWT_EXPIRATION` (default 7200 seconds).

---

### `GET /patient/search`

Searches for patients. **Requires authentication.**

```
Authorization: Bearer <token>
```

Results are always scoped to the hospital in the token — a staff member cannot
reach another hospital's patients by manipulating the query.

**Filters** — all optional, combined with AND:

`national_id` · `passport_id` · `first_name` · `middle_name` · `last_name` ·
`date_of_birth` (`YYYY-MM-DD`) · `phone_number` · `email`

Names match against both the Thai and the English columns, so
`first_name=สมชาย` and `first_name=Somchai` find the same person. Matching is
exact and case-sensitive. Query parameters outside this list are ignored, so a
request carrying only unknown parameters behaves as an unfiltered search.

**How a search resolves:**

1. The local database is queried first.
2. If it returns nothing **and** the request carried a `national_id` or
   `passport_id`, Hospital A's HIS is called, since its API accepts an
   identifier only.
3. The HIS response is validated — it must describe the patient that was
   actually requested — then stored and returned.
4. Without an identifier, the search is answered from local data alone, because
   there is no way to ask the HIS for a name.

| Status | Body |
|---|---|
| `200` | `[ { ...patient }, ... ]` — an empty result is `[]`, never `null` |
| `400` | `{"error":"invalid search filters"}` or `{"error":"unsupported hospital \"X\": only \"A\" has a connected HIS"}` |
| `401` | `{"error":"invalid authorization header"}`, `{"error":"invalid token"}` |
| `404` | `{"error":"patient not found"}` |
| `500` | `{"error":"patient storage failed"}` |
| `502` | `{"error":"Hospital HIS unavailable"}` |

```bash
curl "http://localhost:8080/patient/search?national_id=1234567890123" \
  -H "Authorization: Bearer $TOKEN"
```

```json
[
  {
    "hn": "A0001",
    "national_id": "1234567890123",
    "passport_id": "P1234567",
    "first_name_th": "เดโม",
    "last_name_th": "ผู้ป่วย",
    "first_name_en": "Demo",
    "last_name_en": "Patient",
    "date_of_birth": "1990-01-05",
    "phone_number": "0812345678",
    "email": "demo@example.com",
    "gender": "M"
  }
]
```

Staff of a hospital other than `A` can register and sign in normally — the
assignment defines `hospital` as a free-form field — but no HIS is connected for
them, so a search returns `400` explaining exactly that.

---

## 3. ER Diagram

```
┌─────────────────────────────┐      ┌──────────────────────────────────┐
│ staffs                      │      │ patients                         │
├─────────────────────────────┤      ├──────────────────────────────────┤
│ id             bigserial PK │      │ id              bigserial PK     │
│ username       text  UNIQUE │      │ hospital        text   NOT NULL  │
│ password_hash  text         │      │ hn              text   NOT NULL  │
│ hospital       text         │      │ national_id     text             │
└─────────────────────────────┘      │ passport_id     text             │
                                     │ first_name_th   text             │
   no foreign key between them       │ middle_name_th  text             │
   — see the note below              │ last_name_th    text             │
                                     │ first_name_en   text             │
                                     │ middle_name_en  text             │
                                     │ last_name_en    text             │
                                     │ date_of_birth   date             │
                                     │ phone_number    text             │
                                     │ email           text             │
                                     │ gender          text             │
                                     └──────────────────────────────────┘
```

**Indexes on `patients`**

| Index | Columns | Purpose |
|---|---|---|
| unique | `(hospital, hn)` | one record per patient per hospital; makes the cache write an idempotent upsert |
| index | `(hospital, national_id)` | identifier lookups stay scoped to one hospital |
| index | `(hospital, passport_id)` | same, for passport searches |

Every index leads with `hospital`, because every query in the application is
already scoped to the signed-in staff member's hospital. `national_id` and
`passport_id` are nullable: a patient may carry either identifier or both.

### Why there is no foreign key

The two tables are deliberately independent, linked only by the value in
`hospital`.

`staffs` is data this service owns. `patients` is not — it is a local cache of
records that belong to Hospital A's HIS, kept so that repeated lookups do not
hit the external API. Putting a foreign key between them would couple a cache to
user accounts and block deleting a staff member while cached patients remained,
even though no such relationship exists in the business domain.

### If this had to support more hospitals

The natural next step is a `hospitals` table:

```
hospitals (id, code, name, his_base_url)
     ↑                         ↑
staffs.hospital_id    patients.hospital_id
```

That would move `HOSPITAL_A_BASE_URL` out of the environment and into the
database, so onboarding a second hospital becomes an INSERT rather than a
redeploy, and would let the database itself reject a registration naming a
hospital that does not exist. It is out of scope here: the assignment specifies
`hospital` as a plain field on both APIs, and only one HIS exists.

---

## Notes on the development setup

**`his-a-dummy`** is a small HTTP service in this repository that implements
Hospital A's contract (`GET /patient/search/{id}`) over twelve fixture patients.
It exists because `https://hospital-a.api.co.th` is not reachable, and it means
`docker compose up` produces a stack that can be exercised end to end without
any external dependency. It is reachable only from inside the compose network.

**nginx** is the sole entry point. The application container publishes no port
at all, so traffic genuinely passes through the proxy rather than merely being
able to. The port the application listens on is defined once, in `APP_PORT`, and
compose passes it to nginx, whose configuration is a template rendered at
startup — the proxy target cannot drift from the service.

**Configuration** is read from the environment, with `.env` as a convenience for
local runs. `DATABASE_URL`, `JWT_SECRET`, `JWT_EXPIRATION` and
`HOSPITAL_A_BASE_URL` are required and validated at startup; `APP_ENV`
(`development` or `production`) selects log verbosity and defaults to
`production` so that forgetting it cannot turn on debug output in a deployment.
