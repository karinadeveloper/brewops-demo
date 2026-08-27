ALTER TABLE inventory_movements
    DROP CONSTRAINT inventory_movements_quantity_positive;

ALTER TABLE inventory_movements
    DROP CONSTRAINT inventory_movements_type_check;

ALTER TABLE inventory_movements
    ADD CONSTRAINT inventory_movements_type_check
    CHECK (type IN ('PURCHASE', 'SALE', 'ADJUSTMENT'));
