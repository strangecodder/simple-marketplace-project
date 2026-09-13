DROP INDEX IF EXISTS idx_product_order_user_id;
ALTER TABLE product_order DROP COLUMN IF EXISTS user_id;