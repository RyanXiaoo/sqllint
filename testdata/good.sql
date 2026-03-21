-- This file should pass all lint rules

SELECT id, name, email
FROM users
WHERE active = 1;

DELETE FROM sessions
WHERE expired_at < NOW();

UPDATE users
SET last_login = NOW()
WHERE id = 42;
