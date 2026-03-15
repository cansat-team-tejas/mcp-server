package telemetry

type Record struct {
	ID            int64   `gorm:"column:id;primaryKey;autoIncrement"`
	TeamID        string  `gorm:"column:TEAM_ID"`
	MissionTimeS  float64 `gorm:"column:mission_time_s"`
	PacketCount   int     `gorm:"column:packet_count"`
	Altitude      float64 `gorm:"column:altitude"`
	Pressure      float64 `gorm:"column:pressure"`
	Temperature   float64 `gorm:"column:temperature"`
	Voltage       float64 `gorm:"column:voltage"`
	GNSSTime      string  `gorm:"column:gnss_time"`
	Latitude      float64 `gorm:"column:latitude"`
	Longitude     float64 `gorm:"column:longitude"`
	GPSAltitude   float64 `gorm:"column:gps_altitude"`
	Satellites    int     `gorm:"column:satellites"`
	AccelX        float64 `gorm:"column:accel_x"`
	AccelY        float64 `gorm:"column:accel_y"`
	AccelZ        float64 `gorm:"column:accel_z"`
	GyroSpinRate  float64 `gorm:"column:gyro_spin_rate"`
	FlightState   int     `gorm:"column:flight_state"`
	GyroX         float64 `gorm:"column:gyro_x"`
	GyroY         float64 `gorm:"column:gyro_y"`
	GyroZ         float64 `gorm:"column:gyro_z"`
	Roll          float64 `gorm:"column:roll"`
	Pitch         float64 `gorm:"column:pitch"`
	Yaw           float64 `gorm:"column:yaw"`
	MagX          float64 `gorm:"column:mag_x"`
	MagY          float64 `gorm:"column:mag_y"`
	MagZ          float64 `gorm:"column:mag_z"`
	Humidity      float64 `gorm:"column:humidity"`
	Current       float64 `gorm:"column:current"`
	Power         float64 `gorm:"column:power"`
	BaroAltitude  float64 `gorm:"column:baro_altitude"`
	AirQualityRaw int     `gorm:"column:air_quality_raw"`
	AQEthanolPPM  float64 `gorm:"column:aq_ethanol_ppm"`
	MCUTempC      float64 `gorm:"column:mcu_temp_c"`
	RSSIDBm       int     `gorm:"column:rssi_dbm"`
	HealthFlags   string  `gorm:"column:health_flags"`
	RTCEpoch      int64   `gorm:"column:rtc_epoch"`
	CMDEcho       string  `gorm:"column:cmd_echo"`
	LogData       string  `gorm:"column:log_data"`
}

func (Record) TableName() string {
	return "telemetry"
}
