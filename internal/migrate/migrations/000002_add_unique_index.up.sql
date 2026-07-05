-- Добавляем уникальный индекс на поле original_url
-- Сначала удаляем дубликаты, если они есть
WITH duplicates AS (
    SELECT original_url, MIN(id) as keep_id
    FROM urls
    GROUP BY original_url
    HAVING COUNT(*) > 1
)
DELETE FROM urls
WHERE (original_url, id) NOT IN (
    SELECT original_url, keep_id FROM duplicates
);

-- Теперь создаем уникальный индекс
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_original_url ON urls(original_url);