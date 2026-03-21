-- Implicit joins are harder to read than explicit JOINs

SELECT u.id, o.total
FROM users u, orders o
WHERE u.id = o.user_id;

SELECT u.id, o.total, p.name
FROM users u, orders o, products p
WHERE u.id = o.user_id AND o.product_id = p.id;
