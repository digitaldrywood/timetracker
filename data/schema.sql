-- Time Tracker Database Schema
-- SQLite database for local time tracking

-- Clients table
CREATE TABLE IF NOT EXISTS clients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    rate DECIMAL(10,2) DEFAULT 0,
    currency TEXT DEFAULT 'USD',
    active BOOLEAN DEFAULT 1,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Projects/Repositories table
CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id INTEGER,
    repo_name TEXT NOT NULL UNIQUE,
    description TEXT,
    active BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (client_id) REFERENCES clients(id)
);

-- Time entries table
CREATE TABLE IF NOT EXISTS time_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    date DATE NOT NULL,
    hours DECIMAL(5,2) NOT NULL,
    description TEXT,
    task_type TEXT, -- 'development', 'review', 'meeting', etc.
    billable BOOLEAN DEFAULT 1,
    billed BOOLEAN DEFAULT 0,
    invoice_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (invoice_id) REFERENCES invoices(id)
);

-- Git commits tracking
CREATE TABLE IF NOT EXISTS commits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    sha TEXT NOT NULL,
    message TEXT,
    author TEXT,
    authored_date DATETIME,
    url TEXT,
    time_entry_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (time_entry_id) REFERENCES time_entries(id),
    UNIQUE(project_id, sha)
);

-- Pull requests tracking
CREATE TABLE IF NOT EXISTS pull_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    pr_number INTEGER NOT NULL,
    title TEXT,
    state TEXT, -- 'open', 'closed', 'merged'
    url TEXT,
    created_date DATETIME,
    updated_date DATETIME,
    time_entry_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (time_entry_id) REFERENCES time_entries(id),
    UNIQUE(project_id, pr_number)
);

-- Invoices table
CREATE TABLE IF NOT EXISTS invoices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id INTEGER NOT NULL,
    invoice_number TEXT NOT NULL UNIQUE,
    invoice_date DATE NOT NULL,
    due_date DATE,
    amount DECIMAL(10,2) NOT NULL,
    currency TEXT DEFAULT 'USD',
    status TEXT DEFAULT 'draft', -- 'draft', 'sent', 'paid', 'overdue', 'cancelled'
    paid_date DATE,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (client_id) REFERENCES clients(id)
);

-- Invoice line items
CREATE TABLE IF NOT EXISTS invoice_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    invoice_id INTEGER NOT NULL,
    description TEXT NOT NULL,
    quantity DECIMAL(10,2) DEFAULT 1,
    rate DECIMAL(10,2) NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    time_entry_ids TEXT, -- JSON array of time_entry IDs
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (invoice_id) REFERENCES invoices(id)
);

-- Sync log for Google Sheets synchronization
CREATE TABLE IF NOT EXISTS sync_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_type TEXT NOT NULL, -- 'import', 'export'
    entity_type TEXT NOT NULL, -- 'time_entry', 'invoice', etc.
    entity_id INTEGER,
    spreadsheet_id TEXT,
    sheet_name TEXT,
    row_number INTEGER,
    status TEXT, -- 'success', 'error'
    error_message TEXT,
    synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_time_entries_date ON time_entries(date);
CREATE INDEX idx_time_entries_project ON time_entries(project_id);
CREATE INDEX idx_time_entries_invoice ON time_entries(invoice_id);
CREATE INDEX idx_commits_date ON commits(authored_date);
CREATE INDEX idx_pull_requests_date ON pull_requests(created_date);
CREATE INDEX idx_invoices_client ON invoices(client_id);
CREATE INDEX idx_invoices_status ON invoices(status);

-- Views for common queries
CREATE VIEW IF NOT EXISTS unbilled_time AS
SELECT 
    te.id,
    te.date,
    te.hours,
    te.description,
    p.repo_name as project,
    c.name as client,
    c.rate,
    (te.hours * c.rate) as amount
FROM time_entries te
JOIN projects p ON te.project_id = p.id
JOIN clients c ON p.client_id = c.id
WHERE te.billable = 1 
    AND te.billed = 0
    AND c.active = 1
ORDER BY c.name, te.date;

CREATE VIEW IF NOT EXISTS weekly_summary AS
SELECT 
    strftime('%Y-%W', date) as week,
    c.name as client,
    SUM(te.hours) as total_hours,
    c.rate,
    SUM(te.hours * c.rate) as total_amount
FROM time_entries te
JOIN projects p ON te.project_id = p.id
JOIN clients c ON p.client_id = c.id
WHERE te.billable = 1
GROUP BY week, c.id
ORDER BY week DESC, c.name;