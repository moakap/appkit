package monitoring

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	influxdb "github.com/influxdata/influxdb/client"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/log"
)

// InfluxMonitorConfig type for configuration of Monitor that sinks to
// InfluxDB
type InfluxMonitorConfig string

// NewInfluxdbMonitor creates new monitoring influxdb
// client. config URL syntax is `https://<username>:<password>@<influxDB host>/<database>`
//
// Will returns a error if monitorURL is invalid or not absolute.
//
// Will not return error if InfluxDB is unavailable, but the returned
// Monitor will log errors if it cannot push metrics into InfluxDB
func NewInfluxdbMonitor(config InfluxMonitorConfig, logger log.Logger) (Monitor, error) {
	monitorURL := string(config)

	u, err := url.Parse(monitorURL)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse influxdb url %v", monitorURL)
	} else if !u.IsAbs() {
		return nil, errors.Errorf("influxdb monitoring url %v not absolute url", monitorURL)
	}

	// NewClient always returns a nil error
	client, _ := influxdb.NewClient(influxdb.Config{
		URL: *u,
	})

	monitor := influxdbMonitor{
		database: strings.TrimLeft(u.Path, "/"),
		client:   client,
		logger:   logger,
	}

	logger = logger.With(
		"scheme", u.Scheme,
		"username", u.User.Username(),
		"database", monitor.database,
		"host", u.Host,
	)

	// check connectivity to InfluxDB every 5 minutes
	go func() {
		t := time.NewTimer(5 * time.Minute)

		for {
			// Ignore duration, version
			_, _, err = client.Ping()
			if err != nil {
				logger.Warn().Log(
					"err", err,
					"during", "influxdb.Client.Ping",
					"msg", fmt.Sprintf("couldn't ping influxdb: %v", err),
				)
			}

			<-t.C
		}
	}()

	logger.Info().Log(
		"msg", fmt.Sprintf("influxdb instrumentation writing to %s://%s@%s/%s", u.Scheme, u.User.Username(), u.Host, monitor.database),
	)

	return &monitor, nil
}

// InfluxdbMonitor implements monitor.Monitor interface, it wraps
// the influxdb client configuration.
type influxdbMonitor struct {
	client   *influxdb.Client
	database string
	logger   log.Logger
}

// InsertRecord part of monitor.Monitor.
func (im influxdbMonitor) InsertRecord(measurement string, value interface{}, tags map[string]string, fields map[string]interface{}, at time.Time) {
	if fields == nil {
		fields = map[string]interface{}{}
	}

	fields["value"] = value

	// Ignore response, we only care about write errors
	_, err := im.client.Write(influxdb.BatchPoints{
		Database: im.database,
		Points: []influxdb.Point{
			{
				Measurement: measurement,
				Fields:      fields,
				Tags:        tags,
				Time:        at,
			},
		},
	})

	if err != nil {
		im.logger.Error().Log(
			"err", err,
			"database", im.database,
			"measurement", measurement,
			"value", value,
			"tags", tags,
			"during", "influxdb.Client.Write",
			"msg", fmt.Sprintf("Error inserting record into %s: %v", measurement, err),
		)
	}
}

func (im influxdbMonitor) Count(measurement string, value float64, tags map[string]string, fields map[string]interface{}) {
	im.InsertRecord(measurement, value, tags, fields, time.Now())
}

// CountError logs a value in measurement, with the given error's
// message stored in an `error` tag.
func (im influxdbMonitor) CountError(measurement string, value float64, err error) {
	data := map[string]string{"error": err.Error()}
	im.Count(measurement, value, data, nil)
}

// CountSimple logs a value in measurement (with no tags).
func (im influxdbMonitor) CountSimple(measurement string, value float64) {
	im.Count(measurement, value, nil, nil)
}
