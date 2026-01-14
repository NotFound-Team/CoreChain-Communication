-- name: CreateMeeting :one
INSERT INTO meetings (
    title, description, host_id, room_name, meeting_key, start_time
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;


-- name: AddMeetingInvite :exec
INSERT INTO meeting_invites (meeting_id, user_id)
VALUES ($1, $2);


-- name: GetMeetingByRoomName :one
SELECT * FROM meetings
WHERE room_name = $1 AND end_time IS NULL 
LIMIT 1;


-- name: GetMeetingByID :one
SELECT * FROM meetings
WHERE id = $1 LIMIT 1;


-- name: ListMeetingsForUser :many
SELECT m.* FROM meetings m
INNER JOIN meeting_invites mi ON m.id = mi.meeting_id
WHERE mi.user_id = $1 
  AND m.end_time IS NULL
ORDER BY m.start_time ASC;


-- name: CheckJoinPermission :one
SELECT EXISTS (
    SELECT 1 FROM meetings m
    WHERE m.room_name = sqlc.arg('room_name')
    AND (
        m.host_id = sqlc.arg('user_id')
        OR EXISTS (
            SELECT 1 FROM meeting_invites mi 
            WHERE mi.meeting_id = m.id 
            AND mi.user_id = sqlc.arg('user_id')
        )
    )
    AND m.end_time IS NULL
) AS has_permission;


-- name: GetActiveMeetingByKey :one
SELECT * FROM meetings
WHERE meeting_key = $1 
  AND end_time IS NULL 
LIMIT 1;


-- name: UpdateMeetingStatus :exec
UPDATE meetings
SET is_active = $2
WHERE id = $1;


-- name: EndMeeting :one
UPDATE meetings 
SET is_active = false, end_time = NOW() 
WHERE room_name = $1 AND host_id = $2 AND end_time IS NULL
RETURNING *;

-- name: GetMeetingInvites :many
SELECT user_id FROM meeting_invites
WHERE meeting_id = $1;

-- name: ListMyMeetings :many
SELECT DISTINCT m.*
FROM meetings m
LEFT JOIN meeting_invites mi ON m.id = mi.meeting_id
WHERE (m.host_id = sqlc.arg('user_id') OR mi.user_id = sqlc.arg('user_id'))
  AND m.end_time IS NULL
ORDER BY m.start_time ASC;