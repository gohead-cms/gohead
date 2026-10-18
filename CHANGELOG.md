# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Security
- **Registration privilege escalation**: `POST /auth/register` no longer accepts a client-chosen role. The first account on an empty instance becomes `admin`; later ones are `viewer`.
- **GraphQL authorization bypass**: queries and mutations now enforce role permissions and item validation, like the REST API.
- **JWT**: replaced the unmaintained `dgrijalva/jwt-go` with `golang-jwt/jwt/v5`; only HS256 is accepted and `exp` is required. The sample/empty `jwt_secret` is replaced by a random per-run secret.
- **Agent webhooks**: only `webhook`-triggered agents with a token can be invoked, and the token is compared in constant time.
- **Secrets in responses**: password hashes are never serialized; agent API keys and webhook tokens are masked.
- **Per-agent LLM keys**: keys are passed to each client directly instead of through process-wide environment variables.
- Login no longer logs issued tokens.

### Added
- Admin user management: `/admin/users` (create, list, get, update, delete).
- `GET /admin/components` and `GET /admin/singletons`.
- Filtering (`filters[field][$op]=value`) and multi-key sorting (`sort=field:desc`) on item lists, validated against the collection schema.
- GraphQL `variables` / `operationName`, partial update inputs, list types for to-many relations, and hot reload of the schema when collections change.
- `base_url` option for LLM providers.
- `GOHEAD_TEST_DATABASE_URL` to run the test suite against PostgreSQL.
- Graceful shutdown on SIGINT/SIGTERM.

### Fixed
- Item creation leaked an open database transaction on every request; single and bulk creation are now atomic and events are published after commit.
- Bulk creation stored unvalidated data instead of the processed data.
- GraphQL list pagination passed the offset as a page number.
- Tracing was shut down as soon as the server started; spans are now exported and flushed on exit.
- Agent webhooks crashed because the queue client was never wired into the handler.
- Singleton delete and attribute updates used a non-existent column.
- Restoring a soft-deleted singleton restored another entity's attributes.
- Singleton item updates ignored the singleton's attributes during validation.
- Deleting a component left its attributes behind.
- Roles were duplicated on every startup.
- `JSONMap` could not be scanned from drivers that return strings.
- The GraphQL schema crashed on component/dynamiczone attributes and unknown relation targets.
- `POST /admin/agents` redirected because of a trailing slash.
- Relation population depth is capped (`level` ≤ 5).
- Middleware errors now use the same `{"error": {...}}` envelope as handlers.
- The test suite passes again (7 of 12 packages were failing), including under `-race`.

## [0.1.0] - 2025-01-05
### Added
- **User Role Management**: Introduced roles (`admin`, `editor`, `viewer`) with distinct permissions.
- **API Endpoints**: Implemented `/auth/register` and `/auth/login` routes.
- **Test Utilities**: Added helper methods for mocking JSON POST requests and initializing test servers.
- **Slug Utility**: Added a utility for generating URL-friendly slugs, including support for accented characters.

### Changed
- **Refactored Role Handling**: Centralized role fetching and validation in storage layer.
- **Improved Logging**: Enhanced request and error logging with structured logs for debugging.

### Fixed
- **Database Migrations**: Resolved issues with `AutoMigrate` not applying schema changes during tests.
- **Test Failures**: Fixed inconsistencies in test cases for duplicate entries and invalid roles.

### Legend:
- **Added**: For new features.
- **Changed**: For changes in existing functionality.
- **Fixed**: For any bug fixes.
- **Deprecated**: For soon-to-be removed features.
- **Removed**: For now-removed features.