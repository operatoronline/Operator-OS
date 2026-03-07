-- 011_create_subscriptions: billing plan subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
    id                      TEXT PRIMARY KEY,
    user_id                 TEXT NOT NULL,
    plan_id                 TEXT NOT NULL DEFAULT 'free',
    status                  TEXT NOT NULL DEFAULT 'active',
    billing_interval        TEXT NOT NULL DEFAULT 'none',
    current_period_start    DATETIME NOT NULL,
    current_period_end      DATETIME NOT NULL,
    cancel_at_period_end    INTEGER NOT NULL DEFAULT 0,
    stripe_customer_id      TEXT DEFAULT '',
    stripe_subscription_id  TEXT DEFAULT '',
    created_at              DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at              DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status  ON subscriptions(status);
