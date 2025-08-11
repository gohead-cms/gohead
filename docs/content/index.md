<!-- markdownlint-disable-next-line MD041 -->
[![Build Status](https://github.com/gohead-cms/gohead/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/gohead-cms/gohead/actions/workflows/build.yml?query=branch%3Amain)
[![OpenSSF Best Practices](https://bestpractices.coreinfrastructure.org/projects/XXXX/badge)](https://bestpractices.coreinfrastructure.org/projects/XXXX)
[![Go Report Card](https://goreportcard.com/badge/github.com/gohead-cms/gohead)](https://goreportcard.com/report/github.com/gohead-cms/gohead)
[![License](LICENSE.md)](LICENSE.md)

## What is GoHead?

**GoHead** is a modern, open-source API platform written in Go for managing users, roles, and dynamic JSON-based collections.  
It’s designed for developers building scalable, cloud-native applications, offering **robust authentication**, **flexible validation**, and **database-agnostic storage**—all seamlessly deployable on Kubernetes.

* **User & Role Management** – Authentication, registration, and RBAC built in.
* **Schema-less Collections** – Store structured JSON data without rigid schemas.
* **Field-Level Validation** – Ensure uniqueness, type safety, and data integrity.
* **Kubernetes-Ready** – First-class support for containerized deployments.

---

## Use Cases

- **User Management** for SaaS platforms.
- **Dynamic Data Storage** for CMS-like applications.
- **Backend for Web/Mobile Apps** with minimal setup.
- **Custom APIs** without rebuilding authentication and validation from scratch.

---

## Why GoHead?

- 🪶 **Lightweight & Scalable** – Ideal for modern microservices and edge computing.
- 🔄 **Dynamic JSON Storage** – No migrations for schema changes.
- 🛡 **Role-Based Access Control** – Fine-grained permission system.
- ✅ **Built-in Validation** – Unique field checks, required constraints.
- 🗄 **Database Agnostic** – SQLite (dev), MySQL & PostgreSQL (prod).
- ☸ **Kubernetes Native** – Works out-of-the-box with Ingress, Helm, and cloud deployments.

---

## Try GoHead

1. **Interactive Walkthrough** – Coming soon.
2. **Quickstart Guide** – [docs/quickstart.md](docs/quickstart.md)
3. **Live Demo** – Coming soon at [https://demo.gohead.io](https://demo.gohead.io)

---

## Features

### Core
- **User Management**
  - JWT-based authentication.
  - Role-based permissions.
- **Dynamic Collections**
  - Create and store arbitrary JSON objects.
  - Query and filter without rigid schemas.
- **Observability**
  - Structured logging with Logrus.
  - Integrates with Prometheus & Grafana.

### Advanced
- **Validation Utilities**
  - Unique field checks.
- **Modular Middleware**
  - Auth enforcement, RBAC, and logging.
- **Multi-DB Support**
  - SQLite, MySQL, PostgreSQL drivers built-in.

---

## Quickstart

### Prerequisites
- **Go** 1.21+
- **Docker** (optional)
- **Database**: SQLite, MySQL, or PostgreSQL

### Running Locally
```bash
git clone https://github.com/gohead-cms/gohead.git
cd gohead
cp .env.example .env
go build -o gohead .
./gohead
````

---

## API Endpoints

### User Management

* `POST /auth/register` – Register a new user.
* `POST /auth/login` – Log in & get a JWT.
* `GET /users` – List users.
* `GET /users/:id` – Get user by ID.

### Collections

* `POST /collections` – Create collection.
* `GET /collections/:id` – Get collection.
* `POST /collections/:id/items` – Add item.
* `GET /collections/:id/items` – Get items.

---

## Testing

Run unit tests:

```bash
go test ./...
```

Example:

```go
func TestRegister(t *testing.T) {
    router, _ := testutils.SetupTestServer()
    payload := map[string]string{
        "username": "testuser",
        "password": "securepassword",
        "email": "testuser@example.com",
        "role_name": "viewer",
    }
    body, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", "/auth/register", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusCreated, w.Code)
}
```

---

## Roadmap

* [ ] Interactive demo playground.
* [ ] CLI tool for schema & role management.
* [ ] Built-in WebSocket support.
* [ ] Federation of collections across clusters.

---

## Contributing

We welcome contributions! Please read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting PRs.

---

## Release Cycle

* **Major**: Every 6 months.
* **Minor**: Every 2 months.
* **Patch**: As needed.

Follows [Semantic Versioning](https://semver.org).

---

## Security

Security policy: [SECURITY.md](SECURITY.md)

---

## Credits

* Built with ❤️ by [Nicolas Bounoughaz](https://github.com/sudo-bngz) and the GoHead community.
* Logo design by *to be announced*.