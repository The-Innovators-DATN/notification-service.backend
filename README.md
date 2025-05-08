# Notification Service Documentation

## Introduction

The Notification Service is a backend application designed to manage and send notifications based on events consumed from Kafka. It supports multiple notification channels such as Email and Telegram and provides APIs for managing Contact Points, Policies, and tracking sent Notifications.

### Technologies Used
- Golang
- PostgreSQL
- Kafka
- Gin Web Framework
- SMTP (Email)
- Telegram Bot API

## Installation Requirements

- Golang (>=1.20)
- PostgreSQL
- Kafka
- Docker (optional)

## Project Setup

### Step 1: Clone the repository
```bash
git clone <repository-url>
cd services-service
```

### Step 2: Configure Environment Variables

Create a `.env` file based on the provided template and update it accordingly:

```env
# Kafka configuration
KAFKA_BROKER=
KAFKA_TOPIC=
KAFKA_GROUP_ID=

# PostgreSQL configuration
DB_DSN=

# Email configuration
EMAIL_SMTP_SERVER=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_USERNAME=your-email@gmail.com
EMAIL_PASSWORD=your-email-password
EMAIL_FROM_NAME=

# API configuration
API_PORT=
API_BASE_PATH=

QUEUE_SIZE=
MAX_WORKERS=

# Logging configuration
LOG_LEVEL=
LOG_FILE=
```

### Step 3: Run the Application
```bash
go mod tidy
go run main.go
```

The service runs by default on port `:8080`.

## API Documentation

### Health Check

- **URL**: `/api/v0/health`
- **Method**: `GET`
- **Response**:
```json
{
  "status": "ok"
}
```

### Contact Points

#### Create Contact Point
- **URL**: `/api/v0/contact-points`
- **Method**: `POST`
- **Payload**:
```json
{
  "name": "string",
  "user_id": integer,
  "type": "email|telegram",
  "configuration": "{\"key\":\"value\"}",
  "status": "active|inactive"
}
```
- **Response**:
```json
{
  "success": true,
  "message": "contact point created",
  "data": { /* Contact Point Object */ }
}
```

#### Retrieve Contact Point
- **URL**: `/api/v0/contact-points/:id`
- **Method**: `GET`
- **Response**:
```json
{
  "success": true,
  "message": "contact point retrieved",
  "data": { /* Contact Point Object */ }
}
```

#### List User Contact Points
- **URL**: `/api/v0/contact-points/user/:user_id`
- **Method**: `GET`
- **Response**:
```json
{
  "success": true,
  "message": "contact points list",
  "data": [{ /* Contact Point Object */ }]
}
```

#### Update Contact Point
- **URL**: `/api/v0/contact-points/:id`
- **Method**: `PUT`
- **Payload**: same as Create
- **Response**:
```json
{
  "success": true,
  "message": "contact point updated",
  "data": { /* Updated Contact Point Object */ }
}
```

#### Delete Contact Point
- **URL**: `/api/v0/contact-points/:id`
- **Method**: `DELETE`
- **Response**: HTTP 204 No Content

### Policies

#### Create Policy
- **URL**: `/api/v0/policies`
- **Method**: `POST`
- **Payload**:
```json
{
  "contact_point_id": "UUID",
  "severity": integer,
  "status": "active|inactive",
  "action": "notify",
  "condition_type": "EQ|NEQ|GT|GTE|LT|LTE"
}
```
- **Response**:
```json
{
  "success": true,
  "message": "policy created",
  "data": { /* Policy Object */ }
}
```

#### Retrieve Policy
- **URL**: `/api/v0/policies/:id`
- **Method**: `GET`
- **Response**:
```json
{
  "success": true,
  "message": "policy retrieved",
  "data": { /* Policy Object */ }
}
```

#### List User Policies
- **URL**: `/api/v0/policies/user/:user_id`
- **Method**: `GET`
- **Response**:
```json
{
  "success": true,
  "message": "policies list",
  "data": [{ /* Policy Object */ }]
}
```

#### Update Policy
- **URL**: `/api/v0/policies/:id`
- **Method**: `PUT`
- **Payload**: same as Create
- **Response**:
```json
{
  "success": true,
  "message": "policy updated",
  "data": { /* Updated Policy Object */ }
}
```

#### Delete Policy
- **URL**: `/api/v0/policies/:id`
- **Method**: `DELETE`
- **Response**: HTTP 204 No Content

### Notifications

#### List User Notifications
- **URL**: `/api/v0/notifications/user/:user_id?limit=50&offset=0&status=all`
- **Method**: `GET`
- **Response**: Paginated notification objects

#### List All Notifications
- **URL**: `/api/v0/notifications?limit=50&offset=0&status=all`
- **Method**: `GET`
- **Response**: Paginated notification objects

## Logging

Logs are stored in JSON format with automatic rotation at `./logs/`.

## Kafka Consumer

Processes alerts from Kafka efficiently.

## Notification Providers

- **Email** (SMTP)
- **Telegram** (Bot API)

---

For further support or questions, please contact the development team.

