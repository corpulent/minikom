\connect postgres

CREATE TABLE IF NOT EXISTS events (
    event_id serial PRIMARY KEY,
    service_name VARCHAR ( 50 ) NOT NULL,
    state VARCHAR ( 50 ) NOT NULL,
    dt TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS services (
    service_name VARCHAR ( 50 ) NOT NULL,
    state VARCHAR ( 50 ) NOT NULL,
    dt TIMESTAMP NOT NULL
);
