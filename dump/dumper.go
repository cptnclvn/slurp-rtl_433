package dump

import (
	"fmt"
	"sync"
	"time"

	influxClient "github.com/influxdata/influxdb/client/v2"
	"github.com/jrmycanady/slurp-rtl_433/config"
	"github.com/jrmycanady/slurp-rtl_433/device"
	"github.com/jrmycanady/slurp-rtl_433/logger"
)

type Dumper struct {
	dataPointsChan <-chan device.DataPoint
	cancelChan     chan struct{}
	doneChan       chan struct{}
	cfg            config.Config
	iClient        influxClient.Client
	lock           *sync.Mutex
	running        bool
}

// NewDumper creates a new dumper instance that is ready to start.
func NewDumper(cfg config.Config, dataPointChan <-chan device.DataPoint) *Dumper {
	return &Dumper{
		dataPointsChan: dataPointChan,
		cancelChan:     make(chan struct{}),
		doneChan:       make(chan struct{}),
		cfg:            cfg,
		lock:           &sync.Mutex{},
	}
}

// SetRunning sets the running state of the dumper.
func (d *Dumper) SetRunning(state bool) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.running = state
}

// StartDump attempts to start the dumber. An error is returned if it failed
// to do so.
func (d *Dumper) StartDump() error {

	// Building influxdb client.
	iClient, err := buildInfluxClient(d.cfg)
	if err != nil {
		return err
	}
	d.iClient = iClient

	// Starting dumper process.
	go d.dump()

	return nil
}

// StopDump requests the dumper to stop.
func (d *Dumper) StopDump() {
	d.cancelChan <- struct{}{}
}

// dumper listens on the dataPointsChan and proceses the datapoints as they come
// in.
func (d *Dumper) dump() {
	d.SetRunning(true)
	defer d.SetRunning(false)

	logger.Info.Println("dumper has entered the running state")

	// flushTimerDuration := time.Duration(d.cfg.InfluxDB.FlushTimeTrigger) * time.Second
	// flushTimer := time.NewTimer(flushTimerDuration)

	bp, err := influxClient.NewBatchPoints(influxClient.BatchPointsConfig{
		Database:  d.cfg.InfluxDB.Database,
		Precision: "s",
	})
	if err != nil {
		panic(err)
	}
	for {
		select {
		// case <-flushTimer.C:
		// 	if len(bp.Points()) == 0 {
		// 		continue
		// 	}
		// 	if err := d.iClient.Write(bp); err != nil {
		// 		logger.Error.Printf("failed to send points to InfluxDB: %s", err)
		// 	} else {
		// 		logger.Verbose.Printf("dumped %d datapoints to InfluxDB due to max time reached", len(bp.Points()))
		// 	}
		// 	flushTimer.Reset(time.Duration(flushTimerDuration))
		// 	bp, err = influxClient.NewBatchPoints(influxClient.BatchPointsConfig{
		// 		Database:  d.cfg.InfluxDB.Database,
		// 		Precision: "s",
		// 	})
		// 	if err != nil {
		// 		panic(err)
		// 	}
		case dp := <-d.dataPointsChan:
			switch v := dp.(type) {
			case *device.AmbientWeatherDataPoint:
				p, err := v.InfluxData(d.cfg.Meta[device.AmbientWeatherModelName])
				if err != nil {
					continue
				}

				bp.AddPoint(p)
			}

			// Send if full.
			if len(bp.Points()) >= d.cfg.InfluxDB.FlushDataPointCount {
				if err := d.iClient.Write(bp); err != nil {
					logger.Error.Printf("failed to send points to InfluxDB: %s", err)
				} else {
					logger.Verbose.Printf("dumped %d datapoints to InfluxDB due to max points reached", len(bp.Points()))
				}
				// if !flushTimer.Stop() {
				// 	<-flushTimer.C
				// }
				// flushTimer.Reset(time.Duration(flushTimerDuration))
				bp, err = influxClient.NewBatchPoints(influxClient.BatchPointsConfig{
					Database:  d.cfg.InfluxDB.Database,
					Precision: "s",
				})
				if err != nil {
					panic(err)
				}
			}
		case <-d.cancelChan:
			logger.Info.Println("dumper has received a request to cancel")
			return

		}
	}
}

// buildInfluxClient generates a new InfluxDB client based on the configuration provided.
func buildInfluxClient(config config.Config) (influxClient.Client, error) {
	var err error
	address := ""
	if config.InfluxDB.HTTPS {
		address = fmt.Sprintf("https://%s:%d", config.InfluxDB.FQDN, config.InfluxDB.Port)
	} else {
		address = fmt.Sprintf("http://%s:%d", config.InfluxDB.FQDN, config.InfluxDB.Port)
	}
	client, err := influxClient.NewHTTPClient(influxClient.HTTPConfig{
		Addr:     address,
		Username: config.InfluxDB.Username,
		Password: config.InfluxDB.Password,
	})
	if err != nil {
		return nil, err
	}

	_, _, err = client.Ping(time.Duration(10) * time.Second)
	if err != nil {
		return nil, err
	}

	return client, nil

}
