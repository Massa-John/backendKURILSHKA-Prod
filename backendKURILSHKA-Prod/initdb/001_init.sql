CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    phone_number VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE,
    avatar TEXT,
    password_hash VARCHAR(255) NOT NULL,
    password_changed BOOLEAN DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'offline',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    sender_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    sent_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO users (phone_number, name, email, avatar, password_hash, password_changed, status)
VALUES
    ('001', 'Абонент 001', 'subscriber001@example.com', 'https://api.dicebear.com/7.x/adventurer/svg?seed=001', '12345678', FALSE, 'online'),
    ('002', 'Абонент 002', 'subscriber002@example.com', 'https://api.dicebear.com/7.x/adventurer/svg?seed=002', '12345678', FALSE, 'online')
ON CONFLICT (phone_number) DO NOTHING;

INSERT INTO messages (sender_id, receiver_id, text, sent_at)
SELECT u1.id, u2.id, 'Привет! Как дела?', NOW() - INTERVAL '2 hours'
FROM users u1, users u2
WHERE u1.phone_number = '001' AND u2.phone_number = '002'
ON CONFLICT DO NOTHING;

INSERT INTO messages (sender_id, receiver_id, text, sent_at)
SELECT u1.id, u2.id, 'Всё хорошо! А у тебя?', NOW() - INTERVAL '1 hour'
FROM users u1, users u2
WHERE u1.phone_number = '002' AND u2.phone_number = '001'
ON CONFLICT DO NOTHING;
