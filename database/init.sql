CREATE TABLE IF NOT EXISTS routers (
    id INT AUTO_INCREMENT PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE DEFAULT (UUID()),
    name VARCHAR(100) NOT NULL,
    hostname VARCHAR(45) NOT NULL,
    username VARCHAR(50) NOT NULL,
    password VARCHAR(255) NOT NULL,
    keepalive BOOLEAN DEFAULT TRUE,
    timeout INT DEFAULT 300000,
    port INT DEFAULT 8728,
    location VARCHAR(100),
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    last_seen TIMESTAMP NULL,
    status VARCHAR(20) DEFAULT 'offline',
    version VARCHAR(50),
    uptime VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_hostname (hostname),
    INDEX idx_status (status),
    INDEX idx_is_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO routers (name, username, password, hostname, port) VALUES
('Aeng Panas', 'fandi1', '001', '103.139.193.128', 1012),
('Test', 'admin', 'r00t', '103.139.193.128', 1251);