-- Trading Bot V2 - Initial Schema

CREATE TABLE IF NOT EXISTS config (
    id SERIAL PRIMARY KEY,
    trade_amount_usd DECIMAL(20,8) NOT NULL DEFAULT 100,
    leverage INT NOT NULL DEFAULT 10,
    margin_mode VARCHAR(10) NOT NULL DEFAULT 'ISOLATED',
    commission_rate DECIMAL(10,6) NOT NULL DEFAULT 0.000400,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO config (trade_amount_usd, leverage, margin_mode, commission_rate)
VALUES (100, 10, 'ISOLATED', 0.000400)
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS trades (
    id SERIAL PRIMARY KEY,
    coin VARCHAR(20) NOT NULL,
    signal_type VARCHAR(4) NOT NULL,
    side VARCHAR(5) NOT NULL,
    account_type VARCHAR(12) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    entry_price DECIMAL(20,8) NOT NULL,
    leverage INT NOT NULL,
    commission DECIMAL(20,8) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    binance_order_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_trades_active ON trades(coin, signal_type, is_active);
CREATE INDEX IF NOT EXISTS idx_trades_coin_active ON trades(coin, is_active, signal_type);

CREATE TABLE IF NOT EXISTS pnl_records (
    id SERIAL PRIMARY KEY,
    coin VARCHAR(20) NOT NULL,
    closed_trade_id INT NOT NULL REFERENCES trades(id),
    closing_trade_id INT NOT NULL REFERENCES trades(id),
    closed_signal_type VARCHAR(4) NOT NULL,
    closing_signal_type VARCHAR(4) NOT NULL,
    entry_price DECIMAL(20,8) NOT NULL,
    exit_price DECIMAL(20,8) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    leverage INT NOT NULL,
    gross_pnl DECIMAL(20,8) NOT NULL,
    open_commission DECIMAL(20,8) NOT NULL,
    close_commission DECIMAL(20,8) NOT NULL,
    total_commission DECIMAL(20,8) NOT NULL,
    net_pnl DECIMAL(20,8) NOT NULL,
    side VARCHAR(5) NOT NULL,
    closed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pnl_coin ON pnl_records(coin, closed_at);
CREATE INDEX IF NOT EXISTS idx_pnl_side ON pnl_records(side, closed_at);

CREATE TABLE IF NOT EXISTS webhook_logs (
    id SERIAL PRIMARY KEY,
    request_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    coin VARCHAR(20) NOT NULL,
    signal_type VARCHAR(4) NOT NULL,
    raw_body JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'RECEIVED',
    trade_id INT REFERENCES trades(id),
    error_message TEXT,
    retry_count INT NOT NULL DEFAULT 0,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_webhook_status ON webhook_logs(status, received_at);
CREATE INDEX IF NOT EXISTS idx_webhook_coin ON webhook_logs(coin, signal_type);

CREATE TABLE IF NOT EXISTS execution_logs (
    id SERIAL PRIMARY KEY,
    webhook_id INT NOT NULL REFERENCES webhook_logs(id),
    attempt_number INT NOT NULL DEFAULT 1,
    step VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL,
    account_type VARCHAR(12),
    binance_request JSONB,
    binance_response JSONB,
    error_category VARCHAR(20),
    error_message TEXT,
    duration_ms INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_exec_webhook ON execution_logs(webhook_id, attempt_number);
CREATE INDEX IF NOT EXISTS idx_exec_error ON execution_logs(error_category, status);

CREATE TABLE IF NOT EXISTS binance_accounts (
    id SERIAL PRIMARY KEY,
    account_type VARCHAR(12) NOT NULL UNIQUE,
    api_key TEXT NOT NULL,
    api_secret TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE
);
