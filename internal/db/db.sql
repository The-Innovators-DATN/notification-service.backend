DROP TABLE IF EXISTS alert_dev.notifications;
DROP TABLE IF EXISTS alert_dev.notification_policy;
DROP TABLE IF EXISTS alert_dev.contact_points;

CREATE TABLE alert_dev.notifications
(
    id                     UUID PRIMARY KEY,
    created_at             TIMESTAMP WITH TIME ZONE NOT NULL,
    sent_at                TIMESTAMP WITH TIME ZONE,
    type                   VARCHAR(50)              NOT NULL,
    subject                VARCHAR(255)             NOT NULL,
    body                   TEXT                     NOT NULL,
    notification_policy_id UUID                     NOT NULL,
    status                 VARCHAR(50)              NOT NULL,
    recipient_id           INTEGER                  NOT NULL,
    request_id             VARCHAR(255)             NOT NULL UNIQUE,
    retry_count            INTEGER                  NOT NULL DEFAULT 0,
    last_error             TEXT,
    latest_status          VARCHAR(50)
);

-- Tạo bảng contact_points
CREATE TABLE alert_dev.contact_points
(
    id            UUID PRIMARY KEY,
    name          VARCHAR(255)             NOT NULL,
    user_id       INTEGER                  NOT NULL,
    type          VARCHAR(50)              NOT NULL,
    configuration JSONB                    NOT NULL,
    status        VARCHAR(50)              NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Tạo bảng notification_policy
CREATE TABLE alert_dev.notification_policy
(
    id               UUID PRIMARY KEY,
    contact_point_id UUID                     NOT NULL REFERENCES alert_dev.contact_points (id),
    severity         INTEGER                  NOT NULL,
    status           VARCHAR(50)              NOT NULL,
    topic            VARCHAR(255)             NOT NULL,
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL
);