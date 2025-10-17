package models

import "time"

// Telemetry represents a single telemetry data point from a CanSat mission.
// All fields are nullable to handle incomplete or missing sensor data.
// Data is typically received as CSV format from XBee communication.
type Telemetry struct {
	ID            uint     `gorm:"primaryKey"`
	TeamID        *string  `gorm:"column:TEAM_ID"`
	MissionTimeS  *float64 `gorm:"column:mission_time_s"`
	PacketCount   *int     `gorm:"column:packet_count"`
	Altitude      *float64
	Pressure      *float64
	Temperature   *float64
	Voltage       *float64
	GnssTime      *string `gorm:"column:gnss_time"`
	Latitude      *float64
	Longitude     *float64
	GpsAltitude   *float64 `gorm:"column:gps_altitude"`
	Satellites    *int
	IrnssInView   *int     `gorm:"column:irnss_in_view"`
	IrnssUsed     *int     `gorm:"column:irnss_used"`
	IrnssMask     *int64   `gorm:"column:irnss_mask"`
	AccelX        *float64 `gorm:"column:accel_x"`
	AccelY        *float64 `gorm:"column:accel_y"`
	AccelZ        *float64 `gorm:"column:accel_z"`
	GyroSpinRate  *float64 `gorm:"column:gyro_spin_rate"`
	FlightState   *int     `gorm:"column:flight_state"`
	GyroX         *float64 `gorm:"column:gyro_x"`
	GyroY         *float64 `gorm:"column:gyro_y"`
	GyroZ         *float64 `gorm:"column:gyro_z"`
	Roll          *float64
	Pitch         *float64
	Yaw           *float64
	MagX          *float64 `gorm:"column:mag_x"`
	MagY          *float64 `gorm:"column:mag_y"`
	MagZ          *float64 `gorm:"column:mag_z"`
	Humidity      *float64
	Current       *float64
	Power         *float64
	BaroAltitude  *float64 `gorm:"column:baro_altitude"`
	AirQualityRaw *int     `gorm:"column:air_quality_raw"`
	AqEthanolPpm  *float64 `gorm:"column:aq_ethanol_ppm"`
	McuTempC      *float64 `gorm:"column:mcu_temp_c"`
	RssiDbm       *int     `gorm:"column:rssi_dbm"`
	HealthFlags   *string  `gorm:"column:health_flags"`
	RtcEpoch      *int     `gorm:"column:rtc_epoch"`
	CmdEcho       *string  `gorm:"column:cmd_echo"`
	RwSpeedPct    *int     `gorm:"column:rw_speed_pct"`
	RwSaturated   *int     `gorm:"column:rw_saturated"`
	YawRateTarget *float64 `gorm:"column:yaw_rate_target"`
	PidOutput     *float64 `gorm:"column:pid_output"`
}

func (Telemetry) TableName() string {
	return "telemetry"
}

// ConversationHistory represents a communication message for a specific mission.
// This model tracks all interactions including commands sent to the CanSat,
// telemetry data received, responses, and system errors for complete audit trail.
type ConversationHistory struct {
	ID          uint      `gorm:"primaryKey" json:"id"`                // Auto-generated primary key
	MissionID   string    `gorm:"index;not null" json:"missionId"`     // Mission identifier for grouping
	Timestamp   time.Time `gorm:"not null" json:"timestamp"`           // When the message occurred
	MessageType string    `gorm:"not null" json:"messageType"`         // "command", "telemetry", "response", "error", "log"
	Direction   string    `gorm:"not null" json:"direction"`           // "sent" (from ground) or "received" (from CanSat/system)
	Content     string    `gorm:"type:text" json:"content"`            // The actual message content
	Source      string    `json:"source"`                              // "gui" (ground station), "xbee" (CanSat), "system" (internal)
	Metadata    string    `gorm:"type:json" json:"metadata,omitempty"` // Additional context as JSON string
}
