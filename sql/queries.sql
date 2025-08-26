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
