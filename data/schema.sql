-- Users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    email TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT 1
);

-- Passwords table (separate for security)
CREATE TABLE IF NOT EXISTS passwords (
    username TEXT PRIMARY KEY,
    password_hash TEXT NOT NULL,
    changed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
);

-- SSH keys table
CREATE TABLE IF NOT EXISTS ssh_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    public_key TEXT NOT NULL,
    name TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
);

-- Admin password table
CREATE TABLE IF NOT EXISTS admin_passwords (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT DEFAULT 'admin',
    password_hash TEXT NOT NULL,
    changed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default admin password (admin123)
INSERT INTO admin_passwords (username, password_hash)
VALUES ('admin', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5Mx5WmWbQ7q7a')
ON CONFLICT(username) DO NOTHING;
