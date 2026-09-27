ALTER TABLE orders DROP COLUMN IF EXISTS product_name;
ALTER TABLE client_subscriptions DROP CONSTRAINT IF EXISTS chk_subscription_purchase_price_positive;
ALTER TABLE client_subscriptions DROP COLUMN IF EXISTS purchase_price;
ALTER TABLE client_subscriptions DROP COLUMN IF EXISTS product_name;
