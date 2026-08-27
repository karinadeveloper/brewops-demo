ALTER TABLE inventory_movements
    DROP CONSTRAINT inventory_movements_type_check;

ALTER TABLE inventory_movements
    ADD CONSTRAINT inventory_movements_type_check
    CHECK (type IN ('PURCHASE', 'SALE', 'ADJUSTMENT_IN', 'ADJUSTMENT_OUT'));

ALTER TABLE inventory_movements
    ADD CONSTRAINT inventory_movements_quantity_positive
    CHECK (quantity > 0);
