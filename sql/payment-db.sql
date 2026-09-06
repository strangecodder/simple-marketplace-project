CREATE TABLE balance_ledger (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID NOT NULL,
    is_debit    BOOLEAN NOT NULL,       -- true = дебит (пополнение), false = кредит (списание)
    amount      NUMERIC(18, 2) NOT NULL CHECK (amount > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_balance_ledger_user_id_created_at
    ON balance_ledger (user_id, created_at);