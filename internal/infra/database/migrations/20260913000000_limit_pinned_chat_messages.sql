-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_max_pinned_chat_messages()
RETURNS TRIGGER AS $$
BEGIN
    IF (SELECT COUNT(*) FROM chat_messages WHERE room_id = NEW.room_id AND is_pinned = TRUE) > 20 THEN
        RAISE EXCEPTION 'A room cannot have more than 20 pinned messages.';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_limit_pinned_chat_messages
AFTER INSERT OR UPDATE OF is_pinned ON chat_messages
FOR EACH ROW
WHEN (NEW.is_pinned = TRUE)
EXECUTE FUNCTION check_max_pinned_chat_messages();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_limit_pinned_chat_messages ON chat_messages;
DROP FUNCTION IF EXISTS check_max_pinned_chat_messages;
-- +goose StatementEnd
