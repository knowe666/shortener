CREATE TABLE IF NOT EXISTS urls (
    id          SERIAL PRIMARY KEY,
    short_id    VARCHAR(20) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    user_id     VARCHAR(36) NOT NULL,  -- добавляем поле для идентификатора пользователя
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_short_id ON urls(short_id);
CREATE INDEX idx_original_url ON urls(original_url);
CREATE INDEX idx_user_id ON urls(user_id);  -- индекс для быстрого поиска по пользователю

-- Создаем функцию для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_urls_updated_at BEFORE UPDATE ON urls
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();