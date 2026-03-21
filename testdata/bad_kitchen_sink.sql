-- This file triggers every rule at least once

-- select-star
SELECT * FROM users;

-- keyword-casing (mixed case: Select, From, Where)
Select name, email From accounts Where active = 1;

-- missing-where (DELETE without WHERE)
DELETE FROM sessions;

-- missing-where (UPDATE without WHERE)
UPDATE users SET role = 'admin';

-- leading-wildcard
SELECT id FROM users WHERE name LIKE '%smith';

-- implicit-join
SELECT u.id, o.total FROM users u, orders o WHERE u.id = o.user_id;

-- trailing-semicolon (missing on last statement)
SELECT id
FROM orders
WHERE status = 'pending'
