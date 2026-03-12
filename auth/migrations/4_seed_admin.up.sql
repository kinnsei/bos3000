-- Seed default admin account (password: changeme123)
-- Change this password immediately after first login.
INSERT INTO users (email, username, password_hash, role, status, balance, credit_limit, max_concurrent, daily_limit)
VALUES (
  'admin@localhost',
  'admin',
  '$2a$10$R/XHEykG1mObVZdEwMryfe4r8Nmzy8gdxO5YLVLRlVtcuJjm5RzYS',
  'admin',
  'active',
  0, 0, 0, 0
)
ON CONFLICT (email) DO NOTHING;
