
-- name: CreateConversation :one
INSERT INTO conversations (
    name, 
    avatar,
    is_group
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetConversationByID :one
SELECT 
    c.*, 
    m.content as last_message_content,
    m.sender_id as last_message_sender_id,
    m.type as last_message_type,
    m.file_name as last_message_file_name
FROM conversations c
LEFT JOIN messages m ON c.last_message_id = m.id
WHERE c.id = $1 LIMIT 1;

-- name: UpdateConversationLastMessage :exec
UPDATE conversations
SET 
    last_message_id = $2, 
    last_message_at = $3,
    updated_at = now()
WHERE id = $1;

-- name: UpdateConversationInfo :exec
UPDATE conversations
SET 
    name = $2, 
    avatar = $3,
    updated_at = now()
WHERE id = $1;

-- name: AddParticipant :exec
INSERT INTO participants (
    conversation_id, 
    user_id, 
    role
) VALUES (
    $1, $2, $3
);

-- name: RemoveParticipant :exec
DELETE FROM participants 
WHERE conversation_id = $1 AND user_id = $2;

-- name: ListParticipantsByConversation :many
SELECT user_id, role, joined_at, last_read_message_id
FROM participants 
WHERE conversation_id = $1;


-- name: CreateMessage :one
INSERT INTO messages (
    conversation_id, 
    sender_id, 
    content, 
    type, 
    file_name, 
    file_path, 
    file_type, 
    file_size,
    reply_to_id,
    client_msg_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetMessagesByConversation :many
SELECT * FROM messages
WHERE conversation_id = sqlc.arg('conversation_id')
AND (sqlc.arg('before_id')::bigint = 0 OR id < sqlc.arg('before_id'))
ORDER BY id DESC
LIMIT sqlc.arg('limit_count');

-- name: MarkMessageAsRead :exec
UPDATE participants
SET last_read_message_id = $3
WHERE conversation_id = $1 AND user_id = $2;

-- name: ListConversationsByUser :many
SELECT 
    c.id, 
    c.name, 
    c.avatar,
    c.is_group,
    c.last_message_id,
    c.last_message_at,
    m.content as last_message_content,
    m.sender_id as last_message_sender_id,
    m.type as last_message_type,
    m.file_name as last_message_file_name,
    p.last_read_message_id,
    (
        SELECT COUNT(m2.id) 
        FROM messages m2 
        WHERE m2.conversation_id = c.id 
          AND m2.id > COALESCE(p.last_read_message_id, 0)
          AND m2.sender_id != $1
    ) as unread_count,
    (
        SELECT ARRAY_AGG(user_id)::TEXT[] 
        FROM participants 
        WHERE conversation_id = c.id
    ) as participant_ids
FROM conversations c
JOIN participants p ON c.id = p.conversation_id
LEFT JOIN messages m ON c.last_message_id = m.id
WHERE p.user_id = $1
ORDER BY c.last_message_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPrivateConversation :one
SELECT p1.conversation_id
FROM participants p1
JOIN participants p2 ON p1.conversation_id = p2.conversation_id
JOIN conversations c ON p1.conversation_id = c.id
WHERE c.is_group = FALSE
  AND p1.user_id = $1 
  AND p2.user_id = $2
LIMIT 1;


-- name: UpdateLastReadMessage :exec
UPDATE participants
SET last_read_message_id = $3
WHERE conversation_id = $1 AND user_id = $2;


-- name: GetTotalUnreadCount :one
SELECT COUNT(m.id)
FROM messages m
INNER JOIN participants p ON m.conversation_id = p.conversation_id
WHERE p.user_id = $1 
  AND m.id > COALESCE(p.last_read_message_id, 0)
  AND m.sender_id != $1;

