package commands

type CommandEntry struct {
	Label           string
	Code            string
	Description     string
	Triggers        []string
	KeywordSets     [][]string
	AllowLabelMatch bool
}

var catalog = []CommandEntry{
	{
		Label:           "START_TX",
		Code:            "START_TX",
		Description:     "begin streaming live telemetry packets to the ground station.",
		AllowLabelMatch: true,
		Triggers: []string{
			"start telemetry",
			"start the telemetry",
			"begin telemetry",
			"start telemetry transmission",
			"start transmitting telemetry",
			"enable telemetry transmission",
			"resume telemetry",
			"telemetry on",
			"start tx",
			"enable tx",
		},
		KeywordSets: [][]string{
			{"start", "tx"},
			{"start", "telemetry"},
			{"enable", "tx"},
			{"resume", "telemetry"},
		},
	},
	{
		Label:           "STATUS",
		Code:            "STATUS",
		Description:     "request the current system and mission status report.",
		AllowLabelMatch: true,
		Triggers: []string{
			"get status",
			"status report",
			"system status",
			"mission status",
			"status check",
			"what's the status",
		},
		KeywordSets: [][]string{
			{"status"},
		},
	},
	{
		Label:           "STOP_TX",
		Code:            "STOP_TX",
		Description:     "halt outgoing telemetry packets.",
		AllowLabelMatch: true,
		Triggers: []string{
			"stop telemetry",
			"stop transmitting telemetry",
			"disable telemetry transmission",
			"turn off telemetry",
			"stop tx",
			"halt telemetry",
		},
		KeywordSets: [][]string{
			{"stop", "tx"},
			{"stop", "telemetry"},
			{"disable", "telemetry"},
		},
	},
	{
		Label:           "CAL_SENSORS",
		Code:            "CAL_SENSORS",
		Description:     "run the comprehensive onboard sensor calibration routine.",
		AllowLabelMatch: true,
		Triggers: []string{
			"calibrate sensors",
			"sensor calibration",
			"run calibration",
			"calibrate payload",
			"calibrate the sensors",
		},
		KeywordSets: [][]string{
			{"calibrate", "sensors"},
			{"sensor", "calibration"},
			{"calibration"},
			{"calibrate"},
		},
	},
	{
		Label:           "RESET",
		Code:            "RESET",
		Description:     "perform a system reset back to a safe default state.",
		AllowLabelMatch: true,
		Triggers: []string{
			"reset system",
			"perform reset",
			"reboot system",
			"reset everything",
			"system reset",
		},
		KeywordSets: [][]string{
			{"reset", "system"},
		},
	},
	{
		Label:           "EMERGENCY",
		Code:            "EMERGENCY",
		Description:     "trigger the emergency abort sequence.",
		AllowLabelMatch: true,
		Triggers: []string{
			"emergency",
			"trigger emergency",
			"panic",
			"abort mission",
			"abort sequence",
		},
		KeywordSets: [][]string{
			{"emergency"},
			{"abort", "mission"},
		},
	},
	{
		Label:           "DEPLOY_SECONDARY",
		Code:            "DEPLOY_SECONDARY",
		Description:     "deploy the secondary recovery mechanism (e.g., drogue chute).",
		AllowLabelMatch: true,
		Triggers: []string{
			"deploy secondary",
			"secondary deploy",
			"deploy drogue",
			"deploy backup chute",
			"secondary chute",
		},
		KeywordSets: [][]string{
			{"deploy", "secondary"},
			{"drogue"},
		},
	},
	{
		Label:           "PARACHUTE_DEPLOY",
		Code:            "PARACHUTE_DEPLOY",
		Description:     "command the vehicle to deploy the primary parachute.",
		AllowLabelMatch: true,
		Triggers: []string{
			"deploy parachute",
			"open parachute",
			"parachute deploy",
			"deploy main chute",
		},
		KeywordSets: [][]string{
			{"deploy", "parachute"},
			{"open", "parachute"},
		},
	},
	{
		Label:           "START",
		Code:            "START",
		Description:     "initialize the ground system and prepare subsystems for operation.",
		AllowLabelMatch: false,
		Triggers: []string{
			"start mission",
			"start sequence",
			"system start",
			"start operations",
			"launch start",
			"init ground system",
			"kick off mission",
		},
		KeywordSets: [][]string{
			{"start", "system"},
			{"start", "mission"},
		},
	},
	{
		Label:           "SHUTDOWN",
		Code:            "SHUTDOWN",
		Description:     "shut down the ground system safely.",
		AllowLabelMatch: true,
		Triggers: []string{
			"shutdown",
			"shut down system",
			"power down",
			"terminate system",
			"system shutdown",
		},
		KeywordSets: [][]string{
			{"shut", "down"},
			{"power", "down"},
		},
	},
	{
		Label:           "CLEAR_MISSION",
		Code:            "CLEAR_MISSION",
		Description:     "clear stored mission data from the system.",
		AllowLabelMatch: true,
		Triggers: []string{
			"clear mission",
			"wipe mission",
			"reset mission data",
			"clear mission data",
		},
		KeywordSets: [][]string{
			{"clear", "mission"},
			{"wipe", "mission"},
		},
	},
	{
		Label:           "SD_CLEAN",
		Code:            "SD_CLEAN",
		Description:     "clean up the SD card mission directories.",
		AllowLabelMatch: true,
		Triggers: []string{
			"clean sd",
			"sd clean",
			"clear sd card",
			"clean sd card",
		},
		KeywordSets: [][]string{
			{"clean", "sd"},
		},
	},
	{
		Label:           "SD_INFO",
		Code:            "SD_INFO",
		Description:     "list information about SD card usage.",
		AllowLabelMatch: true,
		Triggers: []string{
			"sd info",
			"sd card info",
			"storage info",
			"storage usage",
		},
		KeywordSets: [][]string{
			{"sd", "info"},
			{"storage", "info"},
		},
	},
	{
		Label:           "SD_LIST",
		Code:            "SD_LIST",
		Description:     "list recordings stored on the SD card.",
		AllowLabelMatch: true,
		Triggers: []string{
			"sd list",
			"list sd files",
			"show sd recordings",
			"list recordings",
		},
		KeywordSets: [][]string{
			{"list", "sd"},
			{"sd", "recordings"},
		},
	},
	{
		Label:           "SD_DIR_INFO",
		Code:            "SD_DIR_INFO:",
		Description:     "show information about a specific SD directory (append the path).",
		AllowLabelMatch: true,
		Triggers: []string{
			"sd dir info",
			"show sd directory",
			"directory info",
			"sd folder info",
		},
		KeywordSets: [][]string{
			{"sd", "directory", "info"},
			{"directory", "info"},
		},
	},
	{
		Label:           "SD_DIR_DELETE",
		Code:            "SD_DIR_DELETE:",
		Description:     "delete the specified directory on the SD card (append the path).",
		AllowLabelMatch: true,
		Triggers: []string{
			"delete sd directory",
			"remove sd directory",
			"delete sd folder",
			"erase sd directory",
		},
		KeywordSets: [][]string{
			{"delete", "sd", "directory"},
			{"remove", "sd", "directory"},
		},
	},
	{
		Label:           "XBEE_RESET",
		Code:            "XBEE_RESET",
		Description:     "reset the XBee radio module via software command.",
		AllowLabelMatch: true,
		Triggers: []string{
			"xbee reset",
			"reset xbee",
			"restart xbee",
		},
		KeywordSets: [][]string{
			{"reset", "xbee"},
		},
	},
	{
		Label:           "XBEE_HW_RESET",
		Code:            "XBEE_HW_RESET",
		Description:     "perform a hardware reset of the XBee radio module.",
		AllowLabelMatch: true,
		Triggers: []string{
			"xbee hardware reset",
			"hardware reset xbee",
			"xbee hard reset",
		},
		KeywordSets: [][]string{
			{"hardware", "reset", "xbee"},
		},
	},
	{
		Label:           "GPS_RESET",
		Code:            "GPS_RESET",
		Description:     "restart the GPS module.",
		AllowLabelMatch: true,
		Triggers: []string{
			"gps reset",
			"reset gps",
			"restart gps",
		},
		KeywordSets: [][]string{
			{"reset", "gps"},
		},
	},
	{
		Label:           "AIR_QUALITY_CAL",
		Code:            "AIR_QUALITY_CAL",
		Description:     "run the air quality sensor calibration routine.",
		AllowLabelMatch: true,
		Triggers: []string{
			"air quality calibration",
			"calibrate air quality",
			"air quality cal",
		},
		KeywordSets: [][]string{
			{"air", "quality", "calibration"},
			{"calibrate", "air", "quality"},
		},
	},
	{
		Label:           "SET_MISSION_START",
		Code:            "SET_MISSION_START",
		Description:     "store the mission start timestamp based on current RTC time.",
		AllowLabelMatch: true,
		Triggers: []string{
			"set mission start",
			"mark mission start",
			"update mission start",
		},
		KeywordSets: [][]string{
			{"set", "mission", "start"},
		},
	},
	{
		Label:           "GET_TIME",
		Code:            "GET_TIME",
		Description:     "retrieve the current RTC timestamp from the vehicle.",
		AllowLabelMatch: true,
		Triggers: []string{
			"get time",
			"current rtc time",
			"what time is it",
			"read rtc",
		},
		KeywordSets: [][]string{
			{"get", "time"},
			{"rtc", "time"},
		},
	},
	{
		Label:           "SET_TIME",
		Code:            "SET_TIME:",
		Description:     "set the RTC to the provided timestamp (append value).",
		AllowLabelMatch: true,
		Triggers: []string{
			"set time",
			"update rtc",
			"set rtc time",
			"adjust rtc",
		},
		KeywordSets: [][]string{
			{"set", "time"},
			{"update", "rtc"},
		},
	},
	{
		Label:           "RTC_STATUS",
		Code:            "RTC_STATUS",
		Description:     "report RTC health and synchronization status.",
		AllowLabelMatch: true,
		Triggers: []string{
			"rtc status",
			"clock status",
			"rtc health",
			"rtc check",
		},
		KeywordSets: [][]string{
			{"rtc", "status"},
			{"clock", "status"},
		},
	},
	{
		Label:           "MCU_STATUS",
		Code:            "MCU_STATUS",
		Description:     "show MCU diagnostics including temperature and load.",
		AllowLabelMatch: true,
		Triggers: []string{
			"mcu status",
			"microcontroller status",
			"board status",
			"mcu health",
		},
		KeywordSets: [][]string{
			{"mcu", "status"},
			{"microcontroller", "status"},
		},
	},
	{
		Label:           "COMM_STATUS",
		Code:            "COMM_STATUS",
		Description:     "report communication link health and signal quality.",
		AllowLabelMatch: true,
		Triggers: []string{
			"comm status",
			"communication status",
			"link status",
			"radio status",
		},
		KeywordSets: [][]string{
			{"communication", "status"},
			{"comm", "status"},
			{"link", "status"},
		},
	},
}

func Catalog() []CommandEntry {
	entries := make([]CommandEntry, len(catalog))
	copy(entries, catalog)
	return entries
}
