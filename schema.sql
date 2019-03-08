-- PostgreSQL schema for flighttrack

-- Version 0: Create database version tracking table
DO
$$
BEGIN
IF NOT EXISTS(SELECT * FROM information_schema.tables
  WHERE table_catalog = CURRENT_CATALOG
  AND table_schema = CURRENT_SCHEMA
  AND table_name = 'schema_version') THEN
    CREATE TABLE schema_version (
      version    INTEGER PRIMARY KEY,
      applied_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
    );
    INSERT INTO schema_version (version) VALUES (0);
END IF;
END;
$$;
-- End Version 0

-- Version 1: Raw message logging table
DO
$$
BEGIN
IF NOT EXISTS(SELECT * FROM schema_version WHERE version >= 1) THEN
  CREATE TABLE raw_message (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    message    BYTEA,
    timestamp  BIGINT,
    signal     SMALLINT,
    created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
  );
  INSERT INTO schema_version (version) VALUES (1);
END IF;
END;
$$;
-- End Version 1