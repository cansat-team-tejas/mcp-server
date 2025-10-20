package database

// TableSchema defines the structure of the telemetry table
// Modify this structure if you need to change the database schema in the future
const TableSchema = `
CREATE TABLE IF NOT EXISTS telemetry (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    TEAM_ID TEXT,
    mission_time_s REAL,
    packet_count INTEGER,
    altitude REAL,
    pressure REAL,
    temperature REAL,
    voltage REAL,
    gnss_time TEXT,
    latitude REAL,
    longitude REAL,
    gps_altitude REAL,
    satellites INTEGER,
    accel_x REAL,
    accel_y REAL,
    accel_z REAL,
    gyro_spin_rate REAL,
    flight_state INTEGER,
    gyro_x REAL,
    gyro_y REAL,
    gyro_z REAL,
    roll REAL,
    pitch REAL,
    yaw REAL,
    mag_x REAL,
    mag_y REAL,
    mag_z REAL,
    humidity REAL,
    current REAL,
    power REAL,
    baro_altitude REAL,
    air_quality_raw INTEGER,
    aq_ethanol_ppm REAL,
    mcu_temp_c REAL,
    rssi_dbm INTEGER,
    health_flags TEXT,
    rtc_epoch INTEGER,
    cmd_echo TEXT,
    log_data TEXT
);
`

// ColumnNames returns all column names for the telemetry table (excluding id)
func ColumnNames() []string {
	return []string{
		"TEAM_ID", "mission_time_s", "packet_count", "altitude", "pressure",
		"temperature", "voltage", "gnss_time", "latitude", "longitude",
		"gps_altitude", "satellites", "accel_x", "accel_y", "accel_z",
		"gyro_spin_rate", "flight_state", "gyro_x", "gyro_y", "gyro_z",
		"roll", "pitch", "yaw", "mag_x", "mag_y", "mag_z",
		"humidity", "current", "power", "baro_altitude", "air_quality_raw",
		"aq_ethanol_ppm", "mcu_temp_c", "rssi_dbm", "health_flags",
		"rtc_epoch", "cmd_echo", "log_data",
	}
}
