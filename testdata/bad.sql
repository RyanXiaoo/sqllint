-- This file has intentional issues for testing the linter

SELECT * FROM users;

Select name, email From users Where active = 1;

DELETE FROM sessions;

UPDATE users SET role = 'admin';

SELECT id, name
FROM orders
WHERE status = 'pending'
ORDER BY created_at DESC;
