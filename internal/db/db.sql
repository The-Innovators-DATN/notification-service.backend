-- Xóa nếu đã tồn tại (theo thứ tự phụ thuộc ngược)
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS notification_policy;
DROP TABLE IF EXISTS contact_points;

-- Bảng contact_points
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

-- Bảng notification_policy
CREATE TABLE IF NOT EXISTS notification_policy (
                                                   id UUID PRIMARY KEY,
                                                   contact_point_id UUID NOT NULL
                                                   REFERENCES contact_points(id)
    ON DELETE CASCADE,
    severity SMALLINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    action VARCHAR(50) NOT NULL,
    condition_type VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- Bảng notifications (kèm ngữ cảnh alert)
CREATE TABLE IF NOT EXISTS notifications (
                                             id UUID PRIMARY KEY,
                                             created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    type VARCHAR(20),
    subject VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    notification_policy_id UUID
    REFERENCES notification_policy(id)
    ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL,
    delivery_method VARCHAR(20),
    recipient_id BIGINT NOT NULL,
    request_id UUID NOT NULL,
    error TEXT,

    -- Alert context fields
    station_id INT,
    metric_id INT,
    metric_name VARCHAR(100),
    operator VARCHAR(20),
    threshold DOUBLE PRECISION,
    threshold_min DOUBLE PRECISION,
    threshold_max DOUBLE PRECISION,
    value DOUBLE PRECISION,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- Indexes tối ưu
CREATE INDEX idx_contact_points_user_id
    ON contact_points(user_id);

CREATE INDEX idx_policy_contact_point_id
    ON notification_policy(contact_point_id);

CREATE INDEX idx_notifications_policy_id
    ON notifications(notification_policy_id);

CREATE INDEX idx_notifications_recipient_id
    ON notifications(recipient_id);

CREATE INDEX idx_notifications_request_id
    ON notifications(request_id);

CREATE INDEX idx_notifications_status
    ON notifications(status);

CREATE INDEX idx_notifications_created_at
    ON notifications(created_at DESC);
