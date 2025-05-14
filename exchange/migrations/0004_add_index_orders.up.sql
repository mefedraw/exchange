CREATE INDEX idx_orders_status_symbol_liq
    ON orders(status, pair_id, liquidation_price);