-- Test: not-in-nullable
-- Should flag: NOT IN with subquery can silently return no rows if any value is NULL
SELECT id, name
FROM users
WHERE id NOT IN (SELECT user_id FROM banned_users);

-- Test: unused-alias
-- Should flag: alias "o" is defined but never referenced
SELECT u.id, u.name
FROM users AS u
JOIN orders AS o ON u.id = o.user_id
WHERE u.active = 1;

-- Test: missing-group-by-col
-- Should flag: "name" is not in GROUP BY and not in an aggregate
SELECT department, name, COUNT(*)
FROM employees
GROUP BY department;
