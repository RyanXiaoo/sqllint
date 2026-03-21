-- Leading wildcards prevent index usage

SELECT id, name
FROM users
WHERE name LIKE '%john';

SELECT id
FROM products
WHERE description ILIKE '%widget%';
