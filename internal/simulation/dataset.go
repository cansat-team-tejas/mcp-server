package simulation

import (
	"context"
	"fmt"
	"math"
	"time"

	"goapp/internal/telemetry"

	"gorm.io/gorm"
)

const (
	teamID        = "1046"
	missionPeriod = 80.0
	tickSeconds   = 0.1
	totalFrames   = 1200
	baseLat       = 13.7199
	baseLon       = 80.2304
	baseEpoch     = int64(1710310800)
)

var stateSymbols = map[int]string{
	2: "{",
	3: "]",
	4: "(",
	5: ")",
	6: "=",
	7: "+",
	8: "*",
}

type phaseInfo struct {
	state    int
	altitude float64
	phase    string
}

func pressureAtAlt(h float64) float64 {
	return 101325 * math.Pow(1-2.25577e-5*h, 5.25588)
}

func tempAtAlt(h float64) float64 {
	return 25 - 6.5*(h/1000)
}

func noise(t, scale, freq, phase float64) float64 { return scale * math.Sin(t*freq+phase) }

func hashNoise(packet int, seed float64) float64 {
	x := math.Sin(float64(packet)*12.9898+seed*78.233) * 43758.5453
	frac := x - math.Floor(x)
	return frac*2 - 1
}

func round(value float64, precision int) float64 {
	factor := math.Pow(10, float64(precision))
	return math.Round(value*factor) / factor
}

func getFlightPhase(t float64) phaseInfo {
	lt := math.Mod(t, missionPeriod)

	if lt < 3 {
		return phaseInfo{state: 0, altitude: 0, phase: "boot"}
	}
	if lt < 5 {
		return phaseInfo{state: 2, altitude: 0, phase: "launch_pad"}
	}
	if lt < 17 {
		frac := (lt - 5) / 12
		return phaseInfo{state: 3, altitude: 850 * (1 - math.Pow(1-frac, 2)), phase: "ascent"}
	}
	if lt < 19 {
		return phaseInfo{state: 4, altitude: 850, phase: "rocket_deploy"}
	}
	if lt < 30 {
		frac := (lt - 19) / 11
		return phaseInfo{state: 5, altitude: 850 - 550*frac, phase: "descent"}
	}
	if lt < 32 {
		return phaseInfo{state: 6, altitude: 300, phase: "secondary_deploy"}
	}
	if lt < 70 {
		frac := (lt - 32) / 38
		return phaseInfo{state: 7, altitude: 300 * (1 - frac), phase: "final_descent"}
	}
	return phaseInfo{state: 8, altitude: 0, phase: "impact"}
}

func healthFlagFor(phase phaseInfo, rssi int, current float64) string {
	if phase.phase == "rocket_deploy" || phase.phase == "secondary_deploy" {
		return "DEPLOY_TRANSIENT"
	}
	if phase.phase == "impact" {
		return "LANDED"
	}
	if rssi < -88 {
		return "LINK_WEAK"
	}
	if phase.phase == "ascent" && current > 2.2 {
		return "BOOST_HIGH_CURRENT"
	}
	return "NOMINAL"
}

func commandEchoFor(packet int, phaseChanged bool, phase phaseInfo) string {
	if packet == 1 {
		return "BOOT"
	}
	if phaseChanged {
		switch phase.phase {
		case "launch_pad":
			return "ARM"
		case "ascent":
			return "LAUNCH"
		case "rocket_deploy":
			return "ROCKET_SEPARATION"
		case "secondary_deploy":
			return "SECONDARY_DEPLOY"
		case "impact":
			return "RECOVERY"
		}
	}
	if packet%300 == 0 {
		return "PING"
	}
	return ""
}

func logDataFor(t float64, phase phaseInfo, phaseChanged bool) string {
	if phaseChanged {
		if sym, ok := stateSymbols[phase.state]; ok {
			return fmt.Sprintf("[%d %s]", int(math.Round(t*100)), sym)
		}
	}
	if int(t*10)%250 == 0 {
		return fmt.Sprintf("[%d #] TRACK", int(math.Round(t*100)))
	}
	return ""
}

func GenerateDataset() []telemetry.Record {
	records := make([]telemetry.Record, 0, totalFrames)
	prevState := -1
	prevAltitude := 0.0
	prevVerticalSpeed := 0.0

	// Correlated low-frequency wind components (m/s)
	windX := 2.2
	windY := -1.1

	for packet := 1; packet <= totalFrames; packet++ {
		t := float64(packet) * tickSeconds
		phase := getFlightPhase(t)
		h := math.Max(0, phase.altitude)
		weatherDrift := noise(t, 1.0, 0.04, 0.25)

		// Update wind with slow random walk + deterministic oscillation.
		windX = 0.985*windX + 0.015*(2.0+0.7*hashNoise(packet, 1.2)) + 0.03*math.Sin(t*0.02)
		windY = 0.985*windY + 0.015*(-1.0+0.6*hashNoise(packet, 2.6)) + 0.03*math.Cos(t*0.017)

		verticalSpeed := (h - prevAltitude) / tickSeconds
		verticalAccel := (verticalSpeed - prevVerticalSpeed) / tickSeconds
		prevAltitude = h
		prevVerticalSpeed = verticalSpeed

		isBoosting := phase.phase == "ascent" && math.Mod(t, missionPeriod) < 8
		isDeploy := phase.phase == "rocket_deploy" || phase.phase == "secondary_deploy"
		isDescent := phase.phase == "descent" || phase.phase == "final_descent"

		lateralBase := 0.08 + 0.015*math.Abs(verticalSpeed)
		if isDeploy {
			lateralBase += 0.5
		}
		accelX := lateralBase*windX + noise(t, 0.2, 1.1, 0.4)
		accelY := lateralBase*windY + noise(t, 0.2, 1.23, 1.2)
		accelZ := 9.81 + 0.25*verticalAccel + noise(t, 0.25, 0.9, 0.2)
		if isBoosting {
			accelZ += 25
		}
		if isDeploy {
			accelZ += noise(t, 4.5, 2.1, 1.1)
		}

		gyroStd := 0.5
		if isDeploy {
			gyroStd = 25
		} else if isBoosting {
			gyroStd = 8
		} else if isDescent {
			gyroStd = 3
		}
		gyroX := noise(t, gyroStd, 1.5, 0.3) + windY*0.7
		gyroY := noise(t, gyroStd, 1.3, 1.4) - windX*0.6
		gyroZ := noise(t, gyroStd*0.6, 1.7, 2.0) + 0.03*verticalSpeed
		spinRate := math.Abs(noise(t, gyroStd, 1.2, 0.5)) + math.Abs(gyroZ)

		roll := noise(t, ternaryFloat(isDeploy, 35, ternaryFloat(isBoosting, 8, 2.5)), 0.8, 0.9)
		pitch := noise(t, ternaryFloat(isDeploy, 30, ternaryFloat(isBoosting, 6, 2.0)), 0.9, 1.1)
		yaw := math.Mod(t*3.0, 360)

		dLat := h*1.2e-6*math.Sin(t*0.08) + windY*2.3e-6*tickSeconds
		dLon := h*1.5e-6*math.Cos(t*0.06) + windX*2.6e-6*tickSeconds
		sats := 0
		if phase.state > 1 {
			sats = int(math.Max(5, math.Round(9+noise(t, 1.5, 0.35, 0.7)-0.0018*h)))
		}

		voltage := 7.38 + noise(t, 0.04, 0.2, 0.4) - float64(packet)*0.00002
		if isBoosting {
			voltage = 7.15 + noise(t, 0.04, 0.2, 0.4)
		}
		current := 0.85 + noise(t, 0.08, 0.6, 0.1)
		if isBoosting {
			current = 2.1 + noise(t, 0.08, 0.6, 0.1)
		}

		pressure := pressureAtAlt(h) + noise(t, 25, 0.5, 0.2)
		temperature := tempAtAlt(h) + weatherDrift + noise(t, 0.4, 0.4, 1.5)
		humidity := math.Max(10, 55-h*0.035) + weatherDrift*0.3 + noise(t, 1.5, 0.4, 0.9)
		mcuTemp := 38 + noise(t, 0.8, 0.5, 0.2)
		if isBoosting {
			mcuTemp += 3
		}

		magX := 28.4 + noise(t, 1.5, 0.3, 0.7)
		magY := -14.8 + noise(t, 1.5, 0.3, 1.1)
		magZ := 41.2 + noise(t, 1.5, 0.3, 2.3)
		rssi := int(math.Round(-54 - h*0.024 + noise(t, 3.5, 0.25, 0.9) - 0.25*math.Abs(windX)))

		epoch := baseEpoch + int64(math.Round(t))
		gnssTime := time.Unix(epoch, 0).UTC().Format("15:04:05")

		phaseChanged := phase.state != prevState
		logData := logDataFor(t, phase, phaseChanged)
		cmdEcho := commandEchoFor(packet, phaseChanged, phase)
		health := healthFlagFor(phase, rssi, current)
		prevState = phase.state

		records = append(records, telemetry.Record{
			TeamID:        teamID,
			MissionTimeS:  round(t, 1),
			PacketCount:   packet,
			Altitude:      round(h, 1),
			Pressure:      round(pressure, 0),
			Temperature:   round(temperature, 1),
			Voltage:       round(voltage, 2),
			GNSSTime:      gnssTime,
			Latitude:      round(baseLat+dLat, 6),
			Longitude:     round(baseLon+dLon, 6),
			GPSAltitude:   round(h+noise(t, 2.5, 0.45, 1.7), 1),
			Satellites:    sats,
			AccelX:        round(accelX, 2),
			AccelY:        round(accelY, 2),
			AccelZ:        round(accelZ, 2),
			GyroSpinRate:  round(spinRate, 2),
			FlightState:   phase.state,
			GyroX:         round(gyroX, 2),
			GyroY:         round(gyroY, 2),
			GyroZ:         round(gyroZ, 2),
			Roll:          round(roll, 1),
			Pitch:         round(pitch, 1),
			Yaw:           round(yaw, 1),
			MagX:          round(magX, 1),
			MagY:          round(magY, 1),
			MagZ:          round(magZ, 1),
			Humidity:      round(humidity, 2),
			Current:       round(current, 2),
			Power:         round(voltage*current, 1),
			BaroAltitude:  round(h+noise(t, 0.8, 0.52, 0.5), 1),
			AirQualityRaw: int(math.Round(180 + h*0.08 + weatherDrift*2 + noise(t, 6, 0.25, 1.3))),
			AQEthanolPPM:  round(0.18+h*0.00052+weatherDrift*0.01+noise(t, 0.03, 0.33, 0.6), 2),
			MCUTempC:      round(mcuTemp, 1),
			RSSIDBm:       rssi,
			HealthFlags:   health,
			RTCEpoch:      epoch,
			CMDEcho:       cmdEcho,
			LogData:       logData,
		})
	}

	return records
}

func SeedDatabase(ctx context.Context, db *gorm.DB) error {
	records := GenerateDataset()

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM telemetry").Error; err != nil {
			return err
		}
		return tx.CreateInBatches(records, 200).Error
	})
}

func ternaryFloat(condition bool, whenTrue, whenFalse float64) float64 {
	if condition {
		return whenTrue
	}
	return whenFalse
}
