# README: Notification Microservice

## Overview

### Project Goal
- **Name**: "Notification Microservice using Go and Kafka, Integrated with Spring Alert System"
- **Purpose**: Build a microservice in Go to receive alert notifications from a Spring-based system via Kafka, process them, and send notifications through multiple channels (email, Telegram, SMS) with an asynchronous retry mechanism, REST API for configuration management, and detailed logging.
- **Key Features**:
    1. Consume messages from Kafka topic `alert_notification` and send notifications.
    2. Handle retries asynchronously with status synchronization (Alert/Resolved).
    3. Provide REST API for managing `contact_points` and `notification_policy`.
    4. Log all activities to a `logs` directory for traceability and debugging.
- **Highlights**:
    - Integration with Spring via Kafka.
    - Non-blocking retry mechanism using Go concurrency.
    - Inspired by Grafana's policy-based notification system, tailored for a graduation project.

---

## Components

- **Input**: JSON messages from Kafka topic `alert_notification`.
- **Output**:
    - Notifications sent via email, Telegram, or SMS.
    - Log files in `logs/YYYY-MM-DD.log`.
    - Status records in PostgreSQL (`notifications` table).
- **Technologies**:
    - **Go**: Core language, leveraging goroutines and channels for concurrency.
    - **Kafka**: Message broker at `160.191.49.50:9092`, topic `alert_notification`.
    - **PostgreSQL**: Database at `160.191.49.50:5432`, credentials `wqSysadmin:asd123`.
    - **Gin**: Framework for REST API.
    - **Providers**: `gomail` (email), `telebot` (Telegram), `twilio-go` (SMS).

---

## Data Structures

### 1. Kafka Message (`AlertNotification`)
- **Source**: Spring application publishing to `alert_notification`.
- **JSON Format**:
  ```json
  {
    "alert_id": "UUID",              // e.g., "123e4567-e89b-12d3-a456-426614174000"
    "alert_name": "string",          // e.g., "Server Down"
    "station_id": int,               // e.g., 1
    "user_id": int,                  // e.g., 1
    "message": "string",             // e.g., "Server not responding"
    "severity": int,                 // 1-4 (low to critical)
    "timestamp": "ISO datetime",     // e.g., "2025-04-10T10:00:00"
    "status": "alert"|"resolved",    // Current status
    "metric_id": int,                // e.g., 101
    "metric_name": "string",         // e.g., "CPU Usage"
    "operator": "string",            // e.g., ">"
    "threshold": double,             // e.g., 90.0
    "threshold_min": double,         // e.g., 0.0
    "threshold_max": double,         // e.g., 100.0
    "value": double                  // e.g., 95.5
  }
  ```
- **Usage**:
    - `alert_id`: Used as `request_id` for tracing.
    - `user_id`: Mapped to `recipient_id`.
    - `severity`: Determines the notification provider via `notification_policy`.
    - `status`: Modifies subject ("Alert: ..." or "Resolved: ...").
    - `alert_name`, `message`, `metric_name`, `value`, `threshold`: Constructs notification content.

### 2. Database (PostgreSQL)
- **Table: `contact_points`**:
    - **Description**: Stores provider configurations for sending notifications.
    - **Columns**:
        - `id`: UUID (Primary Key, e.g., "11111111-1111-1111-1111-111111111111").
        - `name`: VARCHAR(50) (e.g., "Email Provider").
        - `user_id`: BIGINT (e.g., 1).
        - `type`: VARCHAR(20) (Constraint: 'email', 'telegram', 'sms').
        - `configuration`: JSONB (Provider-specific config).
        - `status`: VARCHAR(20) (Constraint: 'active', 'inactive').
        - `created_at`: TIMESTAMPTZ (e.g., "2025-04-10 10:00:00+00").
    - **Sample Data**:
        - Email: `{"addresses": "test@email.com"}`.
        - Telegram: `{"bot_api_token": "xxx", "chat_id": 123456789}`.
        - SMS: `{"phone": "+1234567890", "provider": "Twilio"}`.

- **Table: `notification_policy`**:
    - **Description**: Defines rules for sending notifications based on severity and topic.
    - **Columns**:
        - `id`: UUID (Primary Key, e.g., "44444444-4444-4444-4444-444444444444").
        - `contact_point_id`: UUID (Foreign Key to `contact_points`).
        - `severity`: SMALLINT (Constraint: 1-4).
        - `status`: VARCHAR(20) (Constraint: 'active', 'inactive').
        - `topic`: VARCHAR(50) (e.g., "alert_notification").
        - `retry_max`: SMALLINT (e.g., 3).
        - `retry_interval`: INTERVAL (e.g., '30 seconds').
        - `created_at`: TIMESTAMPTZ.
    - **Sample Data**:
        - Severity 4 → Email, 3 retries, 30 seconds each.
        - Severity 2 → Telegram, 2 retries, 30 seconds each.

- **Table: `notifications`**:
    - **Description**: Tracks notification history and status.
    - **Columns**:
        - `id`: UUID (Primary Key).
        - `created_at`: TIMESTAMPTZ.
        - `sent_at`: TIMESTAMPTZ (NULL if not sent).
        - `type`: VARCHAR(20) (e.g., "alert").
        - `subject`: VARCHAR(255) (e.g., "Alert: Server Down").
        - `body`: TEXT (e.g., "Server not responding\nMetric: CPU Usage\nValue: 95.50\nThreshold: 90.00").
        - `notification_policy_id`: UUID (Foreign Key).
        - `status`: VARCHAR(20) (Constraint: 'pending', 'retrying', 'sent', 'failed', 'cancelled').
        - `recipient_id`: INTEGER (e.g., 1).
        - `request_id`: UUID (e.g., "123e4567-e89b-12d3-a456-426614174000").
        - `retry_count`: SMALLINT (e.g., 0-3).
        - `last_error`: TEXT (e.g., "email timeout").
        - `latest_status`: VARCHAR(20) (Constraint: 'alert', 'resolved', 'cancelled').

---

## Workflow Details

### 1. Kafka Consumer
- **Role**: Receives messages from Kafka, creates tasks, and queues them.
- **Steps**:
    1. Listens to `alert_notification` at `160.191.49.50:9092` with consumer group `notification-group`.
    2. Parses JSON into `AlertNotification` struct.
    3. Generates content:
        - `status = alert`: `subject = "Alert: " + alert_name`.
        - `status = resolved`: `subject = "Resolved: " + alert_name`.
        - `body = message + "\nMetric: " + metric_name + "\nValue: " + value + "\nThreshold: " + threshold`.
    4. Creates task:
        - `topic`: "alert_notification".
        - `severity`: From message.
        - `subject`, `body`: As above.
        - `recipient_id`: `user_id`.
        - `request_id`: `alert_id`.
    5. Pushes task to `tasks` channel (buffer size: 500).

### 2. Worker Pool
- **Role**: Processes initial tasks and sends notifications or queues retries.
- **Quantity**: 10 goroutines.
- **Steps**:
    1. Retrieves task from `tasks`.
    2. Inserts into `notifications`:
        - `status = pending`.
        - `latest_status = alert` or `resolved` (from message).
        - `retry_count = 0`.
    3. Queries `notification_policy`:
        - Condition: `topic = alert_notification` AND `severity = task.severity`.
        - Retrieves `contact_point_id`, then queries `contact_points` for `type` and `configuration`.
    4. Sends initial attempt:
        - Email: Uses `gomail` with `configuration["addresses"]`.
        - Telegram: Uses `telebot` with `bot_api_token` and `chat_id`.
        - SMS: Uses `twilio-go` with `phone`.
    5. Outcome:
        - Success: Updates `status = sent`, `sent_at = now`, logs `[INFO] Sent via {type}`.
        - Failure: Updates `status = retrying`, `retry_count = 1`, `last_error`, pushes to `retryTasks` (buffer size: 500).
    6. Worker becomes available, retrieves next task from `tasks`.

### 3. Retry Manager
- **Role**: Handles retries asynchronously.
- **Quantity**: 2 goroutines (for load balancing).
- **Steps**:
    1. Retrieves task from `retryTasks`.
    2. Waits for retry interval:
        - Uses `<-time.After(30 * time.Second)` (from `notification_policy.retry_interval`).
    3. Checks status:
        - Queries `notifications` with `request_id`, gets latest record:
            - `latest_status = resolved`: Discards task, updates `status = cancelled`, logs `[INFO] Retry cancelled due to resolved`.
            - `latest_status = alert`: Proceeds.
    4. Retries sending:
        - Uses provider as in Worker.
        - Success: Updates `status = sent`, logs `[INFO] Sent via {type}`.
        - Failure: Increments `retry_count`, updates `last_error`:
            - If `retry_count < retry_max`: Pushes back to `retryTasks`.
            - If `retry_count = retry_max`: Updates `status = failed`, logs `[ERROR] Failed after {retry_max} retries`.

### 4. REST API
- **Role**: Manages configuration via HTTP.
- **Port**: `:8080`.
- **Endpoints**:
    - `/contact-points`:
        - **POST**: Creates new contact point → validates JSON → inserts into DB → responds 201 → logs.
        - **GET /:id**: Retrieves details → queries DB → responds 200 → logs.
        - **PUT /:id**: Updates → validates → updates DB → responds 200 → logs.
        - **DELETE /:id**: Deletes → deletes from DB → responds 204 → logs.
    - `/policies`:
        - Similar, with additional validation for `severity`, `retry_max`, `retry_interval`.
- **Security**: Hardcoded basic auth (`admin:123`) for simplicity.
- **Logging**: Logs each request success/failure.

### 5. Logging
- **Directory**: `logs/`.
- **File**: `logs/YYYY-MM-DD.log` (e.g., `logs/2025-04-10.log`).
- **Format**: `[TIMESTAMP] [LEVEL] [request_id=xxx] message`.
- **Levels**:
    - `[INFO]`: Service start, worker start, successful send, API success, retry cancelled.
    - `[ERROR]`: Kafka parse failure, send failure, retry failure, API error.

### 6. Recovery After Crash
- **Job**: Runs every 30 seconds (using `time.Ticker`).
- **Steps**:
    1. Queries `notifications`:
        - Condition: `status = retrying` AND `retry_count < retry_max`.
    2. Creates tasks from records → pushes to `retryTasks`.
    3. Logs: `[INFO] Recovered retry task: {request_id}`.

---

## Use Case Scenarios

### 1. Successful Alert
- **Message**: `"alert_id": "uuid1", "alert_name": "Server Down", "severity": 4, "status": "alert"`.
- **Workflow**:
    1. Consumer → `subject = "Alert: Server Down"`, pushes to `tasks`.
    2. Worker → inserts DB (`status = pending`) → sends email → updates `status = sent`.
    3. Log: `[INFO] [request_id=uuid1] Sent via email`.

### 2. Alert Failure with Retry
- **Message**: As above, but email server is down.
- **Workflow**:
    1. Worker → send fails → updates `status = retrying`, `retry_count = 1`, pushes to `retryTasks`.
    2. Retry Manager → waits 30s → sends again → succeeds → updates `status = sent`.
    3. Log:
       ```
       [ERROR] [request_id=uuid1] Retry queued (1/3): email timeout
       [INFO] [request_id=uuid1] Sent via email
       ```

### 3. Alert Retry Interrupted by Resolved
- **Message 1**: "Alert" at 10:00.
- **Message 2**: "Resolved" at 10:01.
- **Workflow**:
    1. Worker → "Alert" fails → `retry_count = 1`, pushes to `retryTasks`.
    2. Worker → "Resolved" → inserts DB (`latest_status = resolved`) → sends immediately → `status = sent`.
    3. Retry Manager → waits 30s → checks `latest_status = resolved` → updates `status = cancelled`.
    4. Log:
       ```
       [ERROR] [request_id=uuid1] Retry queued (1/3): email timeout
       [INFO] [request_id=uuid1] Resolved sent via email
       [INFO] [request_id=uuid1] Alert retry cancelled due to resolved
       ```

### 4. API Policy Creation
- **Request**: POST `/policies` with `{"severity": 4, "contact_point_id": "1111...", "topic": "alert_notification", "retry_max": 3, "retry_interval": "30 seconds"}`.
- **Workflow**:
    1. API → validates → inserts DB → responds 201.
    2. Log: `[INFO] [request_id=] Created policy: {policy_id}`.

---

## Project Structure
```
notification-service/
├── cmd/
│   └── main.go              // Entry point, starts service, API, and consumer
├── internal/
│   ├── config/             // Loads Kafka, DB credentials from .env
│   ├── db/                // PostgreSQL queries
│   ├── kafka/             // Consumer for alert_notification
│   ├── notification/       // Worker pool and Retry Manager logic
│   ├── api/               // REST API handlers
│   ├── logging/           // File-based logging
│   └── models/            // Structs: AlertNotification, ContactPoint, Policy
├── pkg/                   // Providers: email, telegram, sms
├── logs/                  // Log files
├── .env                   // Environment variables (DB creds, auth)
└── go.mod                 // Go dependencies
```

---

## Implementation Plan

1. **Environment Setup**:
    - Kafka: Connect to `160.191.49.50:9092`, topic `alert_notification`.
    - DB: Connect to `160.191.49.50:5432`, create 3 tables, insert sample data.
    - Go: Initialize project, add dependencies:
        - `confluent-kafka-go`, `pgx/v5`, `gin`, `gomail`, `telebot`, `twilio-go`, `godotenv`.

2. **Database**:
    - Create schema with tables and constraints.
    - Add index on `notifications(request_id, latest_status)` for fast queries.

3. **Configuration**:
    - Load from `.env`: Kafka broker, DB DSN, API auth credentials.

4. **Logging**:
    - Create `logs` directory, write logs to daily files.

5. **Consumer**:
    - Listen to `alert_notification`, parse messages, push to `tasks`.

6. **Worker Pool**:
    - 10 goroutines processing `tasks`, sending initial attempts, queuing retries to `retryTasks`.

7. **Retry Manager**:
    - 2 goroutines processing `retryTasks`, checking `latest_status` before retrying.

8. **API**:
    - Implement CRUD endpoints with basic auth, log all operations.

9. **Recovery**:
    - Periodic job (30s) to recover `retrying` tasks from DB.

---

## Testing & Demo

- **Unit Tests**:
    - Test `notification.Service` with mocked providers.

- **Integration Tests**:
    - Send sample message using `kcat`:
      ```
      kcat -P -b 160.191.49.50:9092 -t alert_notification <<< '{"alert_id": "uuid1", "alert_name": "Server Down", "severity": 4, "status": "alert", "user_id": 1, "message": "Server not responding", "metric_name": "CPU Usage", "value": 95.5, "threshold": 90.0}'
      ```
    - Verify logs, DB records, and provider output (email/Telegram/SMS).

- **Demo**:
    - Script: Trigger Alert → retry failure → retry success → Resolved.
    - Record video with logs and DB screenshots.

---

## Documentation

- **README**:
    - Project description, setup instructions, `.env` configuration.

- **Report**:
    - **Introduction**: Microservice overview, Kafka, Spring integration.
    - **Design**: DB schema, workflow (consumer, worker, retry, API).
    - **Results**: Logs, DB records, demo video.

---

## Additional Notes

- **Debugging**: Regularly check logs for errors and workflow verification.
- **Presentation**: Highlight real-time processing, async retries, Spring integration, and production-like logging.
- **Scalability**: Current design suits a single instance; scale with Kafka partitions if needed later.

---

This README provides a comprehensive blueprint for the Notification Microservice. It details every aspect—data structures, workflows, scenarios, structure, and plans—ensuring a clear path to implementation. Ready to proceed with coding or need further clarification? Let me know!