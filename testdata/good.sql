-- This file should pass all lint rules (Phase 1 + 1.5)

SELECT id, name, email
FROM users
WHERE active = 1;

DELETE FROM sessions
WHERE expired_at < NOW();

UPDATE users
SET last_login = NOW()
WHERE id = 42;

-- Trailing wildcard is fine (index-friendly)
SELECT id, name
FROM users
WHERE name LIKE 'john%';

-- Explicit JOIN instead of comma-separated tables
SELECT u.id, o.total
FROM users AS u
JOIN orders AS o ON u.id = o.user_id
WHERE o.status = 'pending';
