-- +goose Up
-- +goose StatementBegin
alter table chunks add column storage_id integer not null default 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table chunks drop column storage_id;
-- +goose StatementEnd 