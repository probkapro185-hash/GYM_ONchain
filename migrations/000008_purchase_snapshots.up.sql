-- Historical purchases must not change when an admin later edits a product.
-- Do not invent historical product data for malformed legacy rows: stop with a clear
-- error so an operator can repair them before retrying the migration.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM client_subscriptions s
        LEFT JOIN products p ON p.id=s.product_id
        WHERE s.product_id IS NULL OR p.id IS NULL
    ) THEN
        RAISE EXCEPTION 'cannot snapshot subscriptions: legacy row has no valid product_id';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM orders o
        LEFT JOIN products p ON p.id=o.product_id
        WHERE p.id IS NULL
    ) THEN
        RAISE EXCEPTION 'cannot snapshot orders: legacy row has no valid product_id';
    END IF;
END $$;

ALTER TABLE client_subscriptions
    ADD COLUMN product_name VARCHAR(255),
    ADD COLUMN purchase_price NUMERIC(12,2);

UPDATE client_subscriptions s
SET product_name=p.name,
    purchase_price=p.price
FROM products p
WHERE p.id=s.product_id;

ALTER TABLE client_subscriptions
    ALTER COLUMN product_name SET NOT NULL,
    ALTER COLUMN purchase_price SET NOT NULL,
    ADD CONSTRAINT chk_subscription_purchase_price_positive CHECK (purchase_price > 0);

ALTER TABLE orders
    ADD COLUMN product_name VARCHAR(255);

UPDATE orders o
SET product_name=p.name
FROM products p
WHERE p.id=o.product_id;

ALTER TABLE orders
    ALTER COLUMN product_name SET NOT NULL;
