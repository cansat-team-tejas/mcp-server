package api

type AskRequest struct {
	Question string `json:"question"`
}

type QueryRequest struct {
	SQL string `json:"sql"`
}

type TelemetryRow struct {
	TEAM_ID       string  `json:"TEAM_ID"`
	MissionTimeS  float64 `json:"mission_time_s"`
	PacketCount   int     `json:"packet_count"`
	Altitude      float64 `json:"altitude"`
	Pressure      float64 `json:"pressure"`
	Temperature   float64 `json:"temperature"`
	Voltage       float64 `json:"voltage"`
	GNSSTime      string  `json:"gnss_time"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	GPSAltitude   float64 `json:"gps_altitude"`
	Satellites    int     `json:"satellites"`
	AccelX        float64 `json:"accel_x"`
	AccelY        float64 `json:"accel_y"`
	AccelZ        float64 `json:"accel_z"`
	GyroSpinRate  float64 `json:"gyro_spin_rate"`
	FlightState   int     `json:"flight_state"`
	GyroX         float64 `json:"gyro_x"`
	GyroY         float64 `json:"gyro_y"`
	GyroZ         float64 `json:"gyro_z"`
	Roll          float64 `json:"roll"`
	Pitch         float64 `json:"pitch"`
	Yaw           float64 `json:"yaw"`
	MagX          float64 `json:"mag_x"`
	MagY          float64 `json:"mag_y"`
	MagZ          float64 `json:"mag_z"`
	Humidity      float64 `json:"humidity"`
	Current       float64 `json:"current"`
	Power         float64 `json:"power"`
	BaroAltitude  float64 `json:"baro_altitude"`
	AirQualityRaw int     `json:"air_quality_raw"`
	AQEthanolPPM  float64 `json:"aq_ethanol_ppm"`
	MCUTempC      float64 `json:"mcu_temp_c"`
	RSSIDBm       int     `json:"rssi_dbm"`
	HealthFlags   string  `json:"health_flags"`
	RTCEpoch      int64   `json:"rtc_epoch"`
	CMDEcho       string  `json:"cmd_echo"`
	LogData       string  `json:"log_data"`
}

type CreateDatabaseRequest struct {
	DBPath string `json:"db_path"`
}

// TelemetryRowUpper supports payloads that use UPPER_SNAKE_CASE keys (as sent by firmware)
type TelemetryRowUpper struct {
	TEAM_ID        string  `json:"TEAM_ID"`
	MISSION_TIME_S float64 `json:"MISSION_TIME_S"`
	PACKET_COUNT   int     `json:"PACKET_COUNT"`
	ALTITUDE       float64 `json:"ALTITUDE"`
	PRESSURE       float64 `json:"PRESSURE"`
	TEMPERATURE    float64 `json:"TEMPERATURE"`
	VOLTAGE        float64 `json:"VOLTAGE"`
	GNSS_TIME      string  `json:"GNSS_TIME"`
	LATITUDE       float64 `json:"LATITUDE"`
	LONGITUDE      float64 `json:"LONGITUDE"`
	GPS_ALTITUDE   float64 `json:"GPS_ALTITUDE"`
	SATELLITES     int     `json:"SATELLITES"`
	ACCEL_X        float64 `json:"ACCEL_X"`
	ACCEL_Y        float64 `json:"ACCEL_Y"`
	ACCEL_Z        float64 `json:"ACCEL_Z"`
	GYRO_SPIN_RATE float64 `json:"GYRO_SPIN_RATE"`
	FLIGHT_STATE   int     `json:"FLIGHT_STATE"`
	GYRO_X         float64 `json:"GYRO_X"`
	GYRO_Y         float64 `json:"GYRO_Y"`
	GYRO_Z         float64 `json:"GYRO_Z"`
	ROLL           float64 `json:"ROLL"`
	PITCH          float64 `json:"PITCH"`
	YAW            float64 `json:"YAW"`
	MAG_X          float64 `json:"MAG_X"`
	MAG_Y          float64 `json:"MAG_Y"`
	MAG_Z          float64 `json:"MAG_Z"`
	HUMIDITY       float64 `json:"HUMIDITY"`
	CURRENT        float64 `json:"CURRENT"`
	POWER          float64 `json:"POWER"`
	BARO_ALTITUDE  float64 `json:"BARO_ALTITUDE"`
	MCU_TEMP_C     float64 `json:"MCU_TEMP_C"`
	RSSI_DBM       int     `json:"RSSI_DBM"`
	RTC_EPOCH      int64   `json:"RTC_EPOCH"`
	CMD_ECHO       string  `json:"CMD_ECHO"`
	LOG_DATA       string  `json:"LOG_DATA"`
}

// ToLowerCase converts the UPPER_SNAKE payload into our TelemetryRow
func (u TelemetryRowUpper) ToLowerCase() TelemetryRow {
	return TelemetryRow{
		TEAM_ID:      u.TEAM_ID,
		MissionTimeS: u.MISSION_TIME_S,
		PacketCount:  u.PACKET_COUNT,
		Altitude:     u.ALTITUDE,
		Pressure:     u.PRESSURE,
		Temperature:  u.TEMPERATURE,
		Voltage:      u.VOLTAGE,
		GNSSTime:     u.GNSS_TIME,
		Latitude:     u.LATITUDE,
		Longitude:    u.LONGITUDE,
		GPSAltitude:  u.GPS_ALTITUDE,
		Satellites:   u.SATELLITES,
		AccelX:       u.ACCEL_X,
		AccelY:       u.ACCEL_Y,
		AccelZ:       u.ACCEL_Z,
		GyroSpinRate: u.GYRO_SPIN_RATE,
		FlightState:  u.FLIGHT_STATE,
		GyroX:        u.GYRO_X,
		GyroY:        u.GYRO_Y,
		GyroZ:        u.GYRO_Z,
		Roll:         u.ROLL,
		Pitch:        u.PITCH,
		Yaw:          u.YAW,
		MagX:         u.MAG_X,
		MagY:         u.MAG_Y,
		MagZ:         u.MAG_Z,
		Humidity:     u.HUMIDITY,
		Current:      u.CURRENT,
		Power:        u.POWER,
		BaroAltitude: u.BARO_ALTITUDE,
		MCUTempC:     u.MCU_TEMP_C,
		RSSIDBm:      u.RSSI_DBM,
		RTCEpoch:     u.RTC_EPOCH,
		CMDEcho:      u.CMD_ECHO,
		LogData:      u.LOG_DATA,
	}
}
