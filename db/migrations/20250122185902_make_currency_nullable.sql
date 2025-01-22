-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
ALTER COLUMN currency DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
ALTER COLUMN currency SET NOT NULL;
-- +goose StatementEnd
