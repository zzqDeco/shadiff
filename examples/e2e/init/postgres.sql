CREATE TABLE IF NOT EXISTS accounts (
  id INT PRIMARY KEY,
  tier TEXT NOT NULL
);

INSERT INTO accounts (id, tier)
VALUES (1, 'gold')
ON CONFLICT (id) DO UPDATE SET tier = EXCLUDED.tier;
