-- 重置 test@example.com 密码为 test123456
-- bcrypt hash of "test123456" (cost=10)
UPDATE users
SET password_hash = '$2a$10$VPwwyt.Cdnsn0jYrLERBiOe2TcUzeBHT7hZMqqdyfQSSlgwPrB4c2'
WHERE email = 'test@example.com';
