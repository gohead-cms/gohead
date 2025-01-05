# Authentication

This document explains the authentication mechanism used in GoHead, which relies on JWT (JSON Web Tokens) for secure and scalable authentication.

## Overview
GoHead's API uses **JWT Bearer Tokens** for authenticating and authorizing requests. JWT tokens are issued upon successful login and are included in the `Authorization` header of subsequent API requests.

Example:
```http
Authorization: Bearer <your-jwt-token>
```

## JWT Structure
A JWT issued by GoHead contains the following key information:
- **Username**: Identifies the user.
- **Role**: Indicates the user's role, such as `admin`, `editor`, or `viewer`.
- **Standard Claims**:
  - `exp`: Token expiration time (default: 72 hours).
  - `iat`: Time the token was issued.

## Key Functions

### JWT Initialization
Before using JWT, you must initialize it with a secret key:

```go
auth.InitializeJWT("your-secret-key")
```

This secret is used to sign and validate JWT tokens.

### Generate JWT
The `GenerateJWT` function creates a signed token for a given user and their role:

```go
token, err := auth.GenerateJWT(username, role)
if err != nil {
    // Handle error
}
```

### Parse JWT
The `ParseJWT` function validates a token and extracts its claims:

```go
claims, err := auth.ParseJWT(tokenString)
if err != nil {
    // Handle error
}
```

## Authentication Endpoints

### Register
The `Register` endpoint creates a new user in the system.

**Endpoint**:
```http
POST /register
```

**Request Body**:
```json
{
  "username": "example_user",
  "password": "securepassword",
  "email": "example@example.com",
  "role_name": "editor"
}
```

**Response**:
- `201 Created`: User registered successfully.
- `400 Bad Request`: Invalid input or duplicate entry.
- `500 Internal Server Error`: Server error.

### Login
The `Login` endpoint authenticates a user and issues a JWT token.

**Endpoint**:
```http
POST /login
```

**Request Body**:
```json
{
  "username": "example_user",
  "password": "securepassword"
}
```

**Response**:
- `200 OK`: Successful login with a JWT token.
  ```json
  {
    "token": "<jwt-token>"
  }
  ```
- `401 Unauthorized`: Invalid username or password.
- `500 Internal Server Error`: Failed to generate token.

### Token Example
Here’s a decoded example of a JWT token issued by GoHead:

```json
{
  "username": "example_user",
  "role": "editor",
  "exp": 1701234567,
  "iat": 1701230000
}
```

## Security Considerations
- **Keep the secret key secure**: The JWT secret key must be stored securely and not exposed in public repositories.
- **Use HTTPS**: Always serve your API over HTTPS to prevent token interception.
- **Token Expiry**: Enforce short expiration times and provide a mechanism for token refresh.
- **Validate User Roles**: Ensure proper role validation for restricted endpoints.

## Next Steps
- For more information on managing users, refer to the [User Management Overview](user-management/overview.md).
- For advanced configuration, refer to the [Configuration Overview](getting-started/configuration-overview.md).
