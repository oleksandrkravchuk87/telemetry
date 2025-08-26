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
