# Introduction and Overview

The pomodoro backend system provides functionality that runs the single pomodoro at server end with persistence.

The system provides account managmenet that includes autherization, registration, password management, and email verification.

## Requirements

# System Architecture

![System Design Diagram](./docs/architecture/pomodoro-project-backend.svg)

## Rationale

# Data Design

![Data Design Diagram](./docs/data_design/data_design.svg)

# User Interface Design

![User Interface Diagram](./docs/use_case/use_case.svg)

# API

## API Reference

See full OpenAPI spec: [openapi.yaml](./api/openapi.yaml)

---

## Data Models

### Session Type Values

- `WORK`: 25 min work session
- `SHORT_BREAK` : 5 min short break
- `LONG_BREAK` : 20 min long break

---

## Authentication

The API uses JWT Bearer token authentication for protected endpoints. Include the token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```
