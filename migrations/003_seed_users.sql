SET NAMES utf8mb4 COLLATE utf8mb4_0900_ai_ci;

USE app;

INSERT INTO users (id, name, created_at, updated_at, role, is_deleted) VALUES
    (1, '管理者', NOW(), NOW(), 'admin', false),
    (2, '一般ユーザー', NOW(), NOW(), 'user', false)
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    updated_at = VALUES(updated_at),
    role = VALUES(role),
    is_deleted = VALUES(is_deleted);

SET FOREIGN_KEY_CHECKS = 0;

INSERT IGNORE INTO user_ad_account_permissions (user_id, ad_account_id) VALUES
    (1, 'acc_00101'),
    (1, 'acc_00102'),
    (1, 'acc_00103'),
    (1, 'acc_00104'),
    (1, 'acc_00105'),
    (2, 'acc_00106'),
    (2, 'acc_00107'),
    (2, 'acc_00108');

SET FOREIGN_KEY_CHECKS = 1;
