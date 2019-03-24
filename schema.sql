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
IF NOT EXISTS(SELECT * FROM schema_version WHERE version = 1) THEN
  CREATE TABLE raw_message (
    id         INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    message    BYTEA,
    timestamp  BYTEA,
    signal     SMALLINT,
    created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
  );
  INSERT INTO schema_version (version) VALUES (1);
END IF;
END;
$$;
-- End Version 1

-- Version 2: Flight and track log tables
DO
$$
BEGIN
IF NOT EXISTS(SELECT * FROM schema_version WHERE version = 2) THEN
  CREATE TABLE flight (
    id         INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    icao       CHAR(6) NOT NULL,
    callsign   VARCHAR(8),
    first_seen TIMESTAMP NOT NULL,
    last_seen  TIMESTAMP,
    multicall  BOOLEAN DEFAULT FALSE,
    msg_count  INTEGER,
    created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
  );

  CREATE TABLE tracklog (
    id        INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    flight_id INTEGER NOT NULL REFERENCES flight(id),
    time      TIMESTAMP NOT NULL,
    latitude  REAL,
    longitude REAL,
    heading   SMALLINT,
    speed     SMALLINT,
    altitude  INTEGER,
    vs        SMALLINT,
    squak     VARCHAR(4)
  );

  INSERT INTO schema_version (version) VALUES (2);
END IF;
END;
$$;
-- End Version 2