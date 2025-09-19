# TASK-001: Login API

Why
- Authenticate users and return a JWT.

CLI Contract (service-side)
- POST /api/login with fields email and password.
- 200 with a token on success; 401 on invalid credentials.

Acceptance
- Unit and integration tests passing.
- README updated with endpoint usage.

Notes
- Use the existing User model if present.
