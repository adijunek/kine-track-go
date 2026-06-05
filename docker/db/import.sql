-- 1. Import Route Data
COPY osi_routes(id, name, geom, created_at, origin_port_code, dest_port_code, route_type, properties) 
FROM '/docker-entrypoint-initdb.d/data/route.csv' 
WITH (FORMAT csv, HEADER true);

-- 2. Import Cyclone Predictions
-- Added NULL '\N' so Postgres knows how to handle empty cells
COPY cyclone_predictions(id, run_date, run_cycle, forecast_hour, target_time, lat, lon, wind_speed, created_at, geom) 
FROM '/docker-entrypoint-initdb.d/data/cyclone.csv' 
WITH (FORMAT csv, HEADER true, DELIMITER E'\t', NULL '\N');

-- 3. Import Mandalika History
COPY vessel_histories(id, vessel_id, lat, lon, speed, course, status, destination, eta, last_report_at, recorded_at, geom) 
FROM '/docker-entrypoint-initdb.d/data/mandalika.csv' 
WITH (FORMAT csv, HEADER true);