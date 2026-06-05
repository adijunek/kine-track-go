-- Enable PostGIS
CREATE EXTENSION IF NOT EXISTS postgis;

-- 1. Ports Reference Table
CREATE TABLE IF NOT EXISTS osi_ports (
    unlocode TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS osi_routes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    geom GEOMETRY(Geometry, 4326),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    origin_port_code TEXT,
    dest_port_code TEXT,
    route_type TEXT,            -- Added this
    properties JSONB            -- Added this
);

-- 3. Telemetry Table (Expanded to match the 12 columns in mandalika.csv)
CREATE TABLE IF NOT EXISTS vessel_histories (
    id BIGSERIAL PRIMARY KEY,
    vessel_id INTEGER,
    lat NUMERIC(10,7),
    lon NUMERIC(10,7),
    speed NUMERIC,
    course TEXT,
    status TEXT,
    destination TEXT,
    eta TEXT,
    last_report_at TIMESTAMP WITH TIME ZONE,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    geom GEOMETRY(Point, 4326)
);

-- Trigger to keep geom in sync if you receive new JSON pings from Go
CREATE OR REPLACE FUNCTION fn_update_vessel_geom() RETURNS TRIGGER AS $$
BEGIN
    NEW.geom := ST_SetSRID(ST_MakePoint(NEW.lon, NEW.lat), 4326);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_vessel_geom_update 
BEFORE INSERT OR UPDATE ON vessel_histories 
FOR EACH ROW EXECUTE FUNCTION fn_update_vessel_geom();


-- 4. Cyclone Prediction Table (Removed GENERATED ALWAYS so it accepts the 10th CSV column)
CREATE TABLE IF NOT EXISTS cyclone_predictions (
    id BIGSERIAL PRIMARY KEY,
    run_date DATE,
    run_cycle VARCHAR(5),
    forecast_hour BIGINT,
    target_time TIMESTAMP WITH TIME ZONE,
    lat NUMERIC(9,6),
    lon NUMERIC(9,6),
    wind_speed NUMERIC(5,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    geom GEOMETRY(Point, 4326)
);