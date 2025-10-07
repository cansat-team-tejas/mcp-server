package api

type AskRequest struct {
	Question string `json:"question"`
	Filename string `json:"filename"`
}

type QueryRequest struct {
	SQL string `json:"sql"`
}

type CreateDBRequest struct {
	Filename string `json:"filename"`
}

type DataRequest struct {
	Filename string `json:"filename"`
}

type InsertDataRequest struct {
	Filename      string   `json:"filename"`
	TeamID        *string  `json:"TEAM_ID,omitempty"`
	MissionTimeS  *float64 `json:"mission_time_s,omitempty"`
	PacketCount   *int     `json:"packet_count,omitempty"`
	Altitude      *float64 `json:"altitude,omitempty"`
	Pressure      *float64 `json:"pressure,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	Voltage       *float64 `json:"voltage,omitempty"`
	GnssTime      *string  `json:"gnss_time,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	GpsAltitude   *float64 `json:"gps_altitude,omitempty"`
	Satellites    *int     `json:"satellites,omitempty"`
	AccelX        *float64 `json:"accel_x,omitempty"`
	AccelY        *float64 `json:"accel_y,omitempty"`
	AccelZ        *float64 `json:"accel_z,omitempty"`
	GyroSpinRate  *float64 `json:"gyro_spin_rate,omitempty"`
	FlightState   *int     `json:"flight_state,omitempty"`
	GyroX         *float64 `json:"gyro_x,omitempty"`
	GyroY         *float64 `json:"gyro_y,omitempty"`
	GyroZ         *float64 `json:"gyro_z,omitempty"`
	Roll          *float64 `json:"roll,omitempty"`
	Pitch         *float64 `json:"pitch,omitempty"`
	Yaw           *float64 `json:"yaw,omitempty"`
	MagX          *float64 `json:"mag_x,omitempty"`
	MagY          *float64 `json:"mag_y,omitempty"`
	MagZ          *float64 `json:"mag_z,omitempty"`
	Humidity      *float64 `json:"humidity,omitempty"`
	Current       *float64 `json:"current,omitempty"`
	Power         *float64 `json:"power,omitempty"`
	BaroAltitude  *float64 `json:"baro_altitude,omitempty"`
	AirQualityRaw *int     `json:"air_quality_raw,omitempty"`
	AqEthanolPpm  *float64 `json:"aq_ethanol_ppm,omitempty"`
	McuTempC      *float64 `json:"mcu_temp_c,omitempty"`
	RssiDbm       *int     `json:"rssi_dbm,omitempty"`
	HealthFlags   *string  `json:"health_flags,omitempty"`
	RtcEpoch      *int     `json:"rtc_epoch,omitempty"`
	CmdEcho       *string  `json:"cmd_echo,omitempty"`
}
