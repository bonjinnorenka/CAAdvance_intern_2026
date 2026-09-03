SET NAMES utf8mb4 COLLATE utf8mb4_0900_ai_ci;

USE app;

CREATE TABLE IF NOT EXISTS users (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(50),
    created_at DATETIME,
    updated_at DATETIME,
    role VARCHAR(10),
    is_deleted BOOLEAN,

    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS ad_accounts (
    id VARCHAR(20) NOT NULL,
    name VARCHAR(50),
    currency VARCHAR(10),
    status BOOLEAN,
    is_deleted BOOLEAN,

    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS report (
    id INT NOT NULL AUTO_INCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    name VARCHAR(50),
    created_by INT,
    status VARCHAR(10),
    reason VARCHAR(50),
    date_from DATE,
    date_to DATE,
    margin_rate TINYINT,
    file_path VARCHAR(50),
    is_deleted BOOLEAN,

    PRIMARY KEY (id),

    CONSTRAINT fk_report_created_by
        FOREIGN KEY (created_by)
        REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS ad_data (
    ad_account_id VARCHAR(20) NOT NULL,
    date DATE NOT NULL,
    impression INT,
    click INT,
    cost INT,
    conversion INT,

    PRIMARY KEY (ad_account_id, date),

    CONSTRAINT fk_ad_data_ad_account
        FOREIGN KEY (ad_account_id)
        REFERENCES ad_accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user_ad_account_permissions (
    user_id INT NOT NULL,
    ad_account_id VARCHAR(20) NOT NULL,

    PRIMARY KEY (user_id, ad_account_id),

    CONSTRAINT fk_user_ad_account_permissions_user
        FOREIGN KEY (user_id)
        REFERENCES users(id),

    CONSTRAINT fk_user_ad_account_permissions_ad_account
        FOREIGN KEY (ad_account_id)
        REFERENCES ad_accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS report_ad_account (
    report_id INT NOT NULL,
    ad_account_id VARCHAR(20) NOT NULL,

    PRIMARY KEY (report_id, ad_account_id),

    CONSTRAINT fk_report_ad_account_report
        FOREIGN KEY (report_id)
        REFERENCES report(id),

    CONSTRAINT fk_report_ad_account_ad_account
        FOREIGN KEY (ad_account_id)
        REFERENCES ad_accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
