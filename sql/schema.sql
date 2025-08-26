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
