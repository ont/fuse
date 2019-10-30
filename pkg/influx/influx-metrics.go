package influx

import (
	"fmt"
	"fuse/pkg/domain"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
	log "github.com/sirupsen/logrus"
)

type InfluxMetrics struct {
	client   influx.Client // influx api client
	bpConfig influx.BatchPointsConfig
}

func NewInfluxMetrics(opts InfluxOptions) *InfluxMetrics {
	client, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:    opts.Address,
		Timeout: 15 * time.Second,
	})

	if err != nil {
		log.Fatalln("influx: ", err)
	}

	bpConfig := influx.BatchPointsConfig{
		Database:        opts.Database,
		RetentionPolicy: opts.RetentionPolicy,
	}

	return &InfluxMetrics{
		client:   client,
		bpConfig: bpConfig,
	}
}

func (i *InfluxMetrics) Save(msg domain.Message) error {
	bp, err := influx.NewBatchPoints(i.bpConfig)
	if err != nil {
		return err
	}

	tags, values := i.prepareTagValues(msg)

	log.WithField("tags", tags).
		WithField("values", values).Debug("influx: sending 'fuse' metric")

	p, err := influx.NewPoint(
		"fuse", tags, values,
		time.Now(),
	)
	if err != nil {
		return err
	}
	bp.AddPoint(p)
	return i.client.Write(bp)
}

func (i *InfluxMetrics) prepareTagValues(msg domain.Message) (map[string]string, map[string]interface{}) {
	tags := make(map[string]string)
	values := make(map[string]interface{})

	tags["from"] = msg.From
	tags["level"] = msg.LevelToStr()
	tags["title"] = msg.Title
	for k, v := range msg.Details {
		switch k {
		case "value":
			values[k] = v
		default:
			tags[k] = v
		}
	}

	for k, v := range msg.Args {
		tags[k] = fmt.Sprintf("%v", v)
	}

	values["event"] = 1 // always add special value-field for allowing graphs in grafana

	return tags, values
}
