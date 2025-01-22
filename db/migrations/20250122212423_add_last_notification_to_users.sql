-- +goose Up
ALTER TABLE users ADD COLUMN last_notification TIMESTAMP DEFAULT NULL;

-- +goose Down
ALTER TABLE users DROP COLUMN last_notification;
