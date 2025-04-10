-- for now, this basically points to the original instance 
-- while self hosting 
-- as the project grows, i might add in the ability to add 
-- in more instances 
-- and connect to them
-- Placeholder table with the only working functionality being the 
-- base_domain

CREATE TABLE instances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    password TEXT,
    base_domain TEXT,
    is_primary BOOLEAN,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TRIGGER IF EXISTS instances_updated_at;
CREATE TRIGGER instances_updated_at
AFTER UPDATE ON instances
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE instances
        SET updated_at = datetime('now')
        WHERE id = NEW.id;
END;


-- Apps handled by the UI

CREATE TABLE apps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    instance_id INTEGER,
    type TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

create index if not EXISTS idx_apps_name on apps(name);

DROP TRIGGER IF EXISTS apps_updated_at;
CREATE TRIGGER apps_updated_at
AFTER UPDATE ON apps
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE apps
        SET updated_at = datetime('now')
        WHERE id = NEW.id;
END;

CREATE TABLE app_ports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    port TEXT,
    app_id INTEGER NOT NULL,
    domain_id INTEGER,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TRIGGER IF EXISTS apps_updated_at;

CREATE TRIGGER app_ports_updated_at
AFTER UPDATE ON app_ports
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE app_ports
        SET updated_at = datetime('now')
        WHERE id = NEW.id;
END;


-- App Level Domains

CREATE TABLE domains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT,
    app_id INTEGER NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TRIGGER IF EXISTS domains_updated_at;
CREATE TRIGGER domains_updated_at
AFTER UPDATE ON domains
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE domains
        SET updated_at = datetime('now')
        WHERE id = NEW.id;
END;