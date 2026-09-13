-- заглушка для готовых записей, потом исправить
ALTER TABLE product_order
    ADD COLUMN user_id UUID NOT NULL DEFAULT gen_random_uuid();

CREATE INDEX IF NOT EXISTS idx_product_order_user_id ON product_order (user_id);