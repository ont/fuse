package monitor

import (
	"sync"

	"fuse/pkg/domain"
	log "github.com/sirupsen/logrus"
)

type Fuse struct {
	Monitors []Monitor
}

type Monitor interface {
	GetName() string
	RunWith(notifer *domain.Notifer)

	LogInfo()
}

func NewFuse() *Fuse {
	return &Fuse{
		Monitors: make([]Monitor, 0),
	}
}

func (f *Fuse) AddMonitor(monitor Monitor) {
	f.Monitors = append(f.Monitors, monitor)
}

func (f *Fuse) RunWith(notifer *domain.Notifer) {
	var wg sync.WaitGroup

	// TODO: send to another alter if slack is not available
	if notifer.AlerterExists("slack") {
		notifer.Good("slack", domain.Message{
			From:  "fuse",
			Title: "Fuse monitor v0.3.5",
			Body:  "The monitor was restarted",
		})
	}

	notifer.Start()

	wg.Add(len(f.Monitors))
	for _, monitor := range f.Monitors {
		go func(monitor Monitor) {
			log.WithFields(log.Fields{"name": monitor.GetName()}).Info("monitor: starting gorutine")
			defer wg.Done()
			monitor.RunWith(notifer)
		}(monitor)
	}
	wg.Wait()
}
