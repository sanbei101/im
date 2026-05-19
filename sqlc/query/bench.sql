-- name: BatchCreateUser :batchexec
INSERT INTO users (
    user_id, 
    username, 
    password
) VALUES (
    sqlc.arg(user_id), 
    sqlc.arg(username), 
    sqlc.arg(password)
);


-- name: BatchCreateRoom :batchexec
INSERT INTO rooms (
    room_id, 
    chat_type, 
    name,
    avatar_url, 
    single_chat_hash
) VALUES (
    sqlc.arg(room_id), 
    sqlc.arg(chat_type), 
    sqlc.arg(name), 
    sqlc.arg(avatar_url), 
    sqlc.arg(single_chat_hash)
);