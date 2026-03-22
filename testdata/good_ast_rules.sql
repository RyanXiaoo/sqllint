-- Clean: NOT EXISTS instead of NOT IN
SELECT id, name
FROM users
WHERE NOT EXISTS (SELECT 1 FROM banned_users WHERE banned_users.user_id = users.id);

-- Clean: both aliases are referenced
SELECT u.id, o.total
FROM users AS u
JOIN orders AS o ON u.id = o.user_id;

-- Clean: all selected columns are in GROUP BY or aggregated
SELECT department, COUNT(*) AS total
FROM employees
GROUP BY department;
