DROP TABLE IF EXISTS alert_dev.notifications;
DROP TABLE IF EXISTS alert_dev.notification_policy;
DROP TABLE IF EXISTS alert_dev.contact_points;

CREATE TABLE IF NOT EXISTS contact_points (
    id UUID PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    user_id BIGINT NOT NULL,
    type VARCHAR(20) NOT NULL,
    configuration JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS notification_policy (
    id UUID PRIMARY KEY,
    contact_point_id UUID NOT NULL,
    severity SMALLINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    topic VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_contact_point
    FOREIGN KEY (contact_point_id)
    REFERENCES contact_points (id)
    );

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    type VARCHAR(20),
    subject VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    notification_policy_id UUID,
    status VARCHAR(20) NOT NULL,
    recipient_id INT NOT NULL,
    request_id UUID NOT NULL,
    last_error TEXT,
    latest_status VARCHAR(20),
    station_id INT,              -- station_id
    metric_id INT,               -- metric_id
    metric_name VARCHAR(50),     -- metric_name
    operator VARCHAR(10),        -- operator
    threshold DOUBLE PRECISION,  -- threshold
    threshold_min DOUBLE PRECISION, -- threshold_min
    threshold_max DOUBLE PRECISION, -- threshold_max
    value DOUBLE PRECISION,      -- value
    CONSTRAINT fk_policy
    FOREIGN KEY (notification_policy_id)
    REFERENCES notification_policy (id)
    );

CREATE INDEX idx_contact_points_user_id ON contact_points(user_id);

CREATE INDEX idx_notifications_request_id ON notifications(request_id);

CREATE INDEX idx_notifications_recipient_id_status ON notifications(recipient_id);