package influx

// DTO for influx configuration
type InfluxOptions struct {
	Address         string
	Database        string
	RetentionPolicy string
	Interval        int
	Alert           string
}

func DefaultInfluxOptions() InfluxOptions {
	return InfluxOptions{
		Address:         "localhost:8086",
		Database:        "telegraf",
		RetentionPolicy: "",
		Interval:        5,
		Alert:           "",
	}
}
