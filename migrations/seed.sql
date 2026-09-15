-- Sample projects
INSERT INTO projects (name) VALUES
  ('Authentication Service'),
  ('Payment Gateway'),
  ('User Dashboard');

-- Sample scans
INSERT INTO scans (project_id, tool) VALUES
  (1, 'gosec'),
  (1, 'golangci-lint'),
  (2, 'gosec'),
  (2, 'trivy'),
  (3, 'gosec');

-- Sample findings with mixed severities and statuses
INSERT INTO findings (scan_id, title, severity, status, file_path, line_number) VALUES
  (1, 'SQL Injection in login handler', 'critical', 'open', 'internal/auth/handler.go', 42),
  (1, 'Hardcoded API key in config', 'critical', 'resolved', 'config/defaults.go', 15),
  (2, 'Unreachable code after panic', 'low', 'open', 'internal/auth/validator.go', 78),
  (3, 'Weak random number generation', 'high', 'open', 'internal/payment/token.go', 31),
  (4, 'Outdated dependency version', 'medium', 'resolved', 'go.mod', 1),
  (5, 'Missing input validation on user ID', 'high', 'open', 'internal/user/service.go', 105);
