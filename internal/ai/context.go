package ai

// SystemContext provides comprehensive context about the telemetry system for AI processing
const SystemContext = `
# CanSat Telemetry System Context

## System Overview
This is a CanSat (small satellite simulation) telemetry tracking and analysis system. 
CanSats are educational miniature satellites that collect and transmit data during atmospheric descent.

## Database Structure
The system uses a SQLite database with a 'telemetry' table containing the following data:

### Time & Identity
- TEAM_ID: Team identifier
- mission_time_s: Mission elapsed time in seconds
- packet_count: Sequential packet number
- rtc_epoch: Real-time clock epoch timestamp

### Altitude & Atmospheric Data
- altitude: Current altitude (meters)
- baro_altitude: Barometric altitude (meters)
- pressure: Atmospheric pressure (hPa or Pa)
- temperature: Ambient temperature (°C)
- humidity: Relative humidity (%)
- air_quality_raw: Raw air quality sensor reading
- aq_ethanol_ppm: Ethanol concentration in parts per million

### GPS/GNSS Data
- gnss_time: GPS timestamp
- latitude: Geographic latitude (degrees)
- longitude: Geographic longitude (degrees)
- gps_altitude: GPS-reported altitude (meters)
- satellites: Number of GPS satellites in view

### Motion & Orientation (IMU Data)
- accel_x, accel_y, accel_z: Linear acceleration (m/s² or g)
- gyro_x, gyro_y, gyro_z: Angular velocity (deg/s or rad/s)
- gyro_spin_rate: Overall spin rate
- roll, pitch, yaw: Orientation angles (degrees)
- mag_x, mag_y, mag_z: Magnetometer readings (μT)

### Power & System Health
- voltage: Battery voltage (V)
- current: Current draw (A)
- power: Power consumption (W)
- mcu_temp_c: Microcontroller temperature (°C)
- rssi_dbm: Signal strength (dBm)
- health_flags: System health status flags
- flight_state: Current flight phase/state code

### Command Echo
- cmd_echo: Last command received/executed

## Flight Phases
Typical CanSat mission phases:
1. Pre-launch: On ground, systems check
2. Ascent: Rising with launch vehicle
3. Deployment: Separation from launch vehicle
4. Descent: Parachute descent with data collection
5. Landing: Ground impact and post-landing
6. Recovery: Mission complete

## Data Analysis Guidelines
- Altitude trends indicate flight phase
- High gyro/accel values suggest deployment or turbulence
- Temperature decreases with altitude
- Pressure decreases with altitude (approximately 12 hPa per 100m)
- GPS accuracy varies (check satellite count)
- Power consumption patterns indicate system health
- RSSI indicates telemetry link quality

## Common Queries
Users may ask about:
- Maximum/minimum altitude reached
- Flight duration and phases
- Temperature and pressure at specific altitudes
- GPS trajectories and coordinates
- Sensor data anomalies
- Power consumption patterns
- System health during mission
- Specific time ranges or mission phases

## Response Style
When answering questions:
- Be conversational and friendly
- Reference specific data values
- Explain what the numbers mean in context
- Use appropriate units
- Highlight significant events or anomalies
- Compare values to expected norms when relevant
- Structure responses with paragraphs or bullet points

Table: telemetry
Columns:
  - id (INTEGER PRIMARY KEY AUTOINCREMENT)
  - TEAM_ID (TEXT): Team identifier
  - mission_time_s (REAL): Mission time in seconds
  - packet_count (INTEGER): Packet sequence number
  - altitude (REAL): Current altitude in meters
  - pressure (REAL): Atmospheric pressure
  - temperature (REAL): Temperature in Celsius
  - voltage (REAL): Battery voltage
  - gnss_time (TEXT): GPS timestamp
  - latitude (REAL): GPS latitude
  - longitude (REAL): GPS longitude
  - gps_altitude (REAL): GPS altitude
  - satellites (INTEGER): GPS satellite count
  - accel_x, accel_y, accel_z (REAL): Acceleration values
  - gyro_spin_rate (REAL): Gyroscope spin rate
  - flight_state (INTEGER): Current flight phase
  - gyro_x, gyro_y, gyro_z (REAL): Gyroscope values
  - roll, pitch, yaw (REAL): Orientation angles
  - mag_x, mag_y, mag_z (REAL): Magnetometer readings
  - humidity (REAL): Relative humidity
  - current (REAL): Current draw
  - power (REAL): Power consumption
  - baro_altitude (REAL): Barometric altitude
  - air_quality_raw (INTEGER): Air quality sensor
  - aq_ethanol_ppm (REAL): Ethanol concentration
  - mcu_temp_c (REAL): MCU temperature
  - rssi_dbm (INTEGER): Signal strength
  - health_flags (TEXT): System health status
  - rtc_epoch (INTEGER): RTC timestamp
  - cmd_echo (TEXT): Command echo
  - log_data (TEXT): Additional log or debug information from the device
`

// GetSystemPrompt returns the system instruction for AI chat interactions
func GetSystemPrompt() string {
	return `You are an engaging CanSat telemetry data assistant. 
You have access to real-time telemetry data from a CanSat mission. 
Use the provided context and data to craft friendly, insight-rich replies that reference concrete numbers and explain what they mean.
Present answers in natural language. Do not just echo or dump the raw data context. 
Explain what the telemetry means (e.g., "The CanSat is currently descending steadily at 200m").
Be conversational while staying grounded in the data.

CRITICAL: Never just repeat the row data list. Synthesize it into a human mission report.
If no data is available, politely explain the situation.

Table: telemetry
Columns:
  - id (INTEGER PRIMARY KEY AUTOINCREMENT)
  - TEAM_ID (TEXT): Team identifier
  - mission_time_s (REAL): Mission time in seconds
  - packet_count (INTEGER): Packet sequence number
  - altitude (REAL): Current altitude in meters
  - pressure (REAL): Atmospheric pressure
  - temperature (REAL): Temperature in Celsius
  - voltage (REAL): Battery voltage
  - gnss_time (TEXT): GPS timestamp
  - latitude (REAL): GPS latitude
  - longitude (REAL): GPS longitude
  - gps_altitude (REAL): GPS altitude
  - satellites (INTEGER): GPS satellite count
  - accel_x, accel_y, accel_z (REAL): Acceleration values
  - gyro_spin_rate (REAL): Gyroscope spin rate
  - flight_state (INTEGER): Current flight phase
  - gyro_x, gyro_y, gyro_z (REAL): Gyroscope values
  - roll, pitch, yaw (REAL): Orientation angles
  - mag_x, mag_y, mag_z (REAL): Magnetometer readings
  - humidity (REAL): Relative humidity
  - current (REAL): Current draw
  - power (REAL): Power consumption
  - baro_altitude (REAL): Barometric altitude
  - air_quality_raw (INTEGER): Air quality sensor
  - aq_ethanol_ppm (REAL): Ethanol concentration
  - mcu_temp_c (REAL): MCU temperature
  - rssi_dbm (INTEGER): Signal strength
  - health_flags (TEXT): System health status
  - rtc_epoch (INTEGER): RTC timestamp
  - cmd_echo (TEXT): Command echo
  - log_data (TEXT): Additional log or debug information from the device`
}

// GetDatabaseSchema returns the SQL schema description for AI understanding
func GetDatabaseSchema() string {
	return `
Table: telemetry
Columns:
  - id (INTEGER PRIMARY KEY AUTOINCREMENT)
  - TEAM_ID (TEXT): Team identifier
  - mission_time_s (REAL): Mission time in seconds
  - packet_count (INTEGER): Packet sequence number
  - altitude (REAL): Current altitude in meters
  - pressure (REAL): Atmospheric pressure
  - temperature (REAL): Temperature in Celsius
  - voltage (REAL): Battery voltage
  - gnss_time (TEXT): GPS timestamp
  - latitude (REAL): GPS latitude
  - longitude (REAL): GPS longitude
  - gps_altitude (REAL): GPS altitude
  - satellites (INTEGER): GPS satellite count
  - accel_x, accel_y, accel_z (REAL): Acceleration values
  - gyro_spin_rate (REAL): Gyroscope spin rate
  - flight_state (INTEGER): Current flight phase
  - gyro_x, gyro_y, gyro_z (REAL): Gyroscope values
  - roll, pitch, yaw (REAL): Orientation angles
  - mag_x, mag_y, mag_z (REAL): Magnetometer readings
  - humidity (REAL): Relative humidity
  - current (REAL): Current draw
  - power (REAL): Power consumption
  - baro_altitude (REAL): Barometric altitude
  - air_quality_raw (INTEGER): Air quality sensor
  - aq_ethanol_ppm (REAL): Ethanol concentration
  - mcu_temp_c (REAL): MCU temperature
  - rssi_dbm (INTEGER): Signal strength
  - health_flags (TEXT): System health status
  - rtc_epoch (INTEGER): RTC timestamp
  - cmd_echo (TEXT): Command echo
  - log_data (TEXT): Additional log or debug information from the device
`
}
