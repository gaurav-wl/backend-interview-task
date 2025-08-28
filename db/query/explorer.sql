-- name: CreateDecision :exec
INSERT INTO decisions (actor_user_id, recipient_user_id, liked_recipient, created_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (actor_user_id, recipient_user_id)
    DO UPDATE SET
                  liked_recipient = EXCLUDED.liked_recipient,
                  created_at = NOW();

-- name: HasMutualLike :one
SELECT EXISTS(
    SELECT 1 FROM decisions
    WHERE decisions.actor_user_id = $1 AND decisions.recipient_user_id = $2 AND decisions.liked_recipient = true
) AND EXISTS(
    SELECT 1 FROM decisions
    WHERE decisions.actor_user_id = $2 AND decisions.recipient_user_id = $1 AND decisions.liked_recipient = true
);

-- name: CountLikes :one
SELECT COUNT(*)
FROM decisions
WHERE recipient_user_id = $1 AND liked_recipient = true;
