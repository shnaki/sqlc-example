ALTER TABLE authors ADD COLUMN email text;

-- NULL は UNIQUE 制約に含まれないため複数の NULL 行が共存できる。
-- ON CONFLICT (email) DO UPDATE は UNIQUE 制約がある列でのみ使える。
ALTER TABLE authors ADD CONSTRAINT authors_email_unique UNIQUE (email);
