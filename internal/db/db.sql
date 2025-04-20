-- Drop tables if they exist (in reverse order of dependencies)
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS notification_policy;
DROP TABLE IF EXISTS contact_points;

-- Create contact_points table
CREATE TABLE IF NOT EXISTS contact_points (
    id UUID PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    user_id BIGINT NOT NULL,
    type VARCHAR(20) NOT NULL,
    configuration JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- Create notification_policy table
CREATE TABLE IF NOT EXISTS notification_policy (
    id UUID PRIMARY KEY,
    contact_point_id UUID NOT NULL,
    severity SMALLINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    condition_type VARCHAR(50),
    CONSTRAINT fk_contact_point
    FOREIGN KEY (contact_point_id)
    REFERENCES contact_points (id)
    );

-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    type VARCHAR(20),
    subject VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    notification_policy_id UUID,
    status VARCHAR(20) NOT NULL,
    delivery_method VARCHAR(20),
    recipient_id BIGINT NOT NULL,
    request_id UUID NOT NULL,
    error TEXT,

    CONSTRAINT fk_policy
    FOREIGN KEY (notification_policy_id)
    REFERENCES notification_policy (id)
    );

-- Create indexes for better query performance
CREATE INDEX idx_contact_points_user_id ON contact_points(user_id);
CREATE INDEX idx_notifications_request_id ON notifications(request_id);
CREATE INDEX idx_notifications_recipient_id ON notifications(recipient_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);