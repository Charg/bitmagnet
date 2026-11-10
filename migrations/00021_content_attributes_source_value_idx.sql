-- +goose Up
-- +goose StatementBegin

-- Looks up content by an alternative identifier (e.g. an IMDb ID) without scanning every attribute of the source.
CREATE INDEX IF NOT EXISTS content_attributes_source_value_idx ON content_attributes (source, value);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS content_attributes_source_value_idx;

-- +goose StatementEnd
