SET NAMES utf8mb4 COLLATE utf8mb4_0900_ai_ci;

USE app;

UPDATE users
SET role = 'member', updated_at = NOW()
WHERE role = 'user';
