-- +goose Up
-- +goose StatementBegin
create table chunks (
    uuid text not null,
    chunk_index bigint not null,
    chunk_hash text not null,
    status text not null,
    updated_at timestamp not null default now(),
    num_of_chunks bigint not null,
    constraint chunks_pkey primary key (uuid, chunk_index)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table chunks;
-- +goose StatementEnd
