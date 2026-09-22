CREATE TABLE projects (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE scans (
    id          BIGSERIAL PRIMARY KEY,
    project_id  BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tool        TEXT NOT NULL,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE findings (
    id          BIGSERIAL PRIMARY KEY,
    scan_id     BIGINT NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    severity    TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'confirmed', 'false_positive', 'resolved')),
    file_path   TEXT NOT NULL,
    line_number INTEGER NOT NULL CHECK (line_number > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_scans_project_id ON scans(project_id);
CREATE INDEX idx_findings_scan_id ON findings(scan_id);
CREATE INDEX idx_findings_severity ON findings(severity);
CREATE INDEX idx_findings_status ON findings(status);
