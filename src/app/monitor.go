package main

import log "github.com/sirupsen/logrus"
import "sync"

type Fuse struct {
    Monitors []Monitor
}

type Monitor interface {
    GetName() string
    RunWith(notifer *Notifer)
}

func NewFuse() *Fuse {
    return &Fuse{
        Monitors: make([]Monitor, 0),
    }
}

func (f *Fuse) AddMonitor(monitor Monitor){
    f.Monitors = append(f.Monitors, monitor)
}

func (f *Fuse) RunWith(notifer *Notifer) {
    var wg sync.WaitGroup

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
