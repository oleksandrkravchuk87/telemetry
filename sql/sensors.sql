-- Schema MySQL
CREATE TABLE IF NOT EXISTS rooms (
                                     id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
                                     name VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS sensors (
                                       id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
                                       name VARCHAR(100) UNIQUE NOT NULL,
                                       room_id INT UNSIGNED NOT NULL,
                                       type CHAR(1) NOT NULL CHECK (type IN ('V', 'R')),
                                       FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS measurements (
                                            id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
                                            sensor_id INT UNSIGNED NOT NULL,
                                            value FLOAT NOT NULL,
                                            timestamp TIMESTAMP(6) NOT NULL,
                                            FOREIGN KEY (sensor_id) REFERENCES sensors(id) ON DELETE CASCADE
);

CREATE INDEX idx_measurements_sensor_time ON measurements(sensor_id, timestamp);


INSERT INTO rooms (name) VALUES ('room_A');
INSERT INTO rooms (name) VALUES ('room_B');

-- The room room_A has 1 sensor for V and 2 sensors for R
INSERT INTO sensors (name, room_id, type)
SELECT vals.name, r.id, vals.type
FROM (
         SELECT 'room_A_V1' AS name, 'V' AS type
         UNION ALL
         SELECT 'room_A_R1', 'R'
         UNION ALL
         SELECT 'room_A_R2', 'R'
     ) AS vals
         JOIN rooms r ON r.name = 'room_A';

-- The room room_B has 2 sensors for V and 3 sensors for R
INSERT INTO sensors (name, room_id, type)
SELECT vals.name, r.id, vals.type
FROM (
         SELECT 'room_B_V1' AS name, 'V' AS type
         UNION ALL
         SELECT 'room_B_V2', 'V'
         UNION ALL
         SELECT 'room_B_R1', 'R'
         UNION ALL
         SELECT 'room_B_R2', 'R'
         UNION ALL
         SELECT 'room_B_R3', 'R'
     ) AS vals
         JOIN rooms r ON r.name = 'room_B';

-- measurements from 2 of 3 sensors
INSERT INTO measurements (sensor_id, value, timestamp)
SELECT s.id, vals.value, vals.ts
FROM (
         SELECT 'room_A_V1' AS sensor_name, 50.0 AS value, '2025-08-24 10:00:00.000001' AS ts
         UNION ALL
         SELECT 'room_A_R1', 100.0, '2025-08-24 10:00:00.000002'
     ) AS vals
         JOIN sensors s ON s.name = vals.sensor_name;

-- room_A, only R present, no V
INSERT INTO measurements (sensor_id, value, timestamp)
VALUES (
           (SELECT id FROM sensors WHERE name = 'room_A_R2'),
           101.0,
           '2025-08-24 10:00:05.000001'
       );

-- room_B, two V sensors, three R sensors, all present
INSERT INTO measurements (sensor_id, value, timestamp)
SELECT s.id, vals.value, vals.ts
FROM (
         SELECT 'room_B_V1' AS sensor_name, 49.9 AS value, '2025-08-24 10:00:10.000010' AS ts
         UNION ALL
         SELECT 'room_B_V2', 50.2, '2025-08-24 10:10:00.000010'
         UNION ALL
         SELECT 'room_B_R1', 99.8, '2025-08-24 10:10:00.000020'
         UNION ALL
         SELECT 'room_B_R2', 100.1, '2025-08-24 10:10:00.000030'
         UNION ALL
         SELECT 'room_B_R3', 100.0, '2025-08-24 10:10:00.000040'
     ) AS vals
         JOIN sensors s ON s.name = vals.sensor_name;

-- Select Query 1 (grouped by room and second, average values)
SELECT
    a.room AS room,
    a.ts AS timestamp,
    ROUND(a.v_avg, 2) AS V,
    ROUND(a.r_avg, 2) AS R,
    ROUND(a.v_avg / NULLIF(a.r_avg, 0), 2) AS I
FROM (
    SELECT
    r.name AS room,
    CAST(m.timestamp AS DATETIME) AS ts,
    AVG(CASE WHEN s.type = 'V' THEN m.value END) AS v_avg,
    AVG(CASE WHEN s.type = 'R' THEN m.value END) AS r_avg
    FROM
    measurements m
    JOIN sensors s ON m.sensor_id = s.id
    JOIN rooms r ON s.room_id = r.id
    GROUP BY
    r.name,
    CAST(m.timestamp AS DATETIME)
    ) a
ORDER BY
    a.room, a.ts;


-- Select Query 2 (use the R value from the previous second)
WITH v_agg AS (
    SELECT
        r.name AS room,
        CAST(DATE_FORMAT(m.timestamp, '%Y-%m-%d %H:%i:%s') AS DATETIME) AS ts,
        AVG(m.value) AS v_avg
    FROM measurements m
             JOIN sensors s ON m.sensor_id = s.id
             JOIN rooms r ON s.room_id = r.id
    WHERE s.type = 'V'
    GROUP BY r.name, ts
),

-- AVG R for each second
     r_avg AS (
         SELECT
             r.name AS room,
             CAST(DATE_FORMAT(m.timestamp, '%Y-%m-%d %H:%i:%s') AS DATETIME) AS ts,
             AVG(m.value) AS r_avg
         FROM measurements m
                  JOIN sensors s ON m.sensor_id = s.id
                  JOIN rooms r ON s.room_id = r.id
         WHERE s.type = 'R'
         GROUP BY r.name, ts
     ),

-- Last R for each second (timestamp)
     r_raw AS (
         SELECT
             r.name AS room,
             m.value,
             m.timestamp,
             CAST(DATE_FORMAT(m.timestamp, '%Y-%m-%d %H:%i:%s') AS DATETIME) AS ts
         FROM measurements m
                  JOIN sensors s ON m.sensor_id = s.id
                  JOIN rooms r ON s.room_id = r.id
         WHERE s.type = 'R'
     ),

     r_latest_per_sec AS (
         SELECT
             room,
             ts,
             value,
             ROW_NUMBER() OVER (PARTITION BY room, ts ORDER BY timestamp DESC) AS rn
         FROM r_raw
     ),

     r_last AS (
         SELECT room, ts, value
         FROM r_latest_per_sec
         WHERE rn = 1
     ),

-- For each V get r_avg, or last R from prev second
     r_selected AS (
         SELECT
             v.room,
             v.ts AS v_ts,
             COALESCE(
                     (SELECT r_avg.r_avg FROM r_avg WHERE r_avg.room = v.room AND r_avg.ts = v.ts),
                     (SELECT r_last.value FROM r_last WHERE r_last.room = v.room AND r_last.ts = v.ts - INTERVAL 1 SECOND)
             ) AS r_value
         FROM v_agg v
     )

SELECT
    v.room,
    v.ts AS timestamp,
    ROUND(v.v_avg, 2) AS V,
    ROUND(r.r_value, 2) AS R,
    ROUND(v.v_avg / NULLIF(r.r_value, 0), 2) AS I
FROM v_agg v
         LEFT JOIN r_selected r
                   ON v.room = r.room AND v.ts = r.v_ts
ORDER BY v.room, v.ts;
