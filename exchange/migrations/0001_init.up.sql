CREATE TYPE order_type AS ENUM ('long', 'short');
CREATE TYPE order_status AS ENUM ('open', 'closed', 'liquidated', 'canceled');

CREATE TABLE users
(
    id        BIGSERIAL PRIMARY KEY,
    email     VARCHAR(255)   NOT NULL UNIQUE,
    pass_hash TEXT           NOT NULL,
    balance   DECIMAL(20, 2) NOT NULL DEFAULT 0,
    created   TIMESTAMPTZ    NOT NULL
);


CREATE TABLE trading_pairs
(
    id          BIGSERIAL PRIMARY KEY,
    base_asset  VARCHAR(10) NOT NULL,
    quote_asset VARCHAR(10) NOT NULL,
    CONSTRAINT unique_pair UNIQUE (base_asset, quote_asset)
);


CREATE TABLE orders
(
    id          UUID PRIMARY KEY NOT NULL,
    user_id     BIGINT           NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    pair_id     BIGINT           NOT NULL REFERENCES trading_pairs (id) ON DELETE CASCADE,
    type        order_type       NOT NULL,
    margin      DECIMAL(20, 2)   NOT NULL,
    leverage    SMALLINT         NOT NULL,
    entry_price DECIMAL(20, 2)   NOT NULL,
    close_price DECIMAL(20, 2),
    status      order_status     NOT NULL,
    created_at  TIMESTAMP        NOT NULL
);