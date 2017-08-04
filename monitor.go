package main

import "sync"

type Fuse struct {
    Monitors []Monitor
}

type Monitor interface {
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
        go func() {
            monitor.RunWith(notifer)
            defer wg.Done()
        }()
    }
    wg.Wait()
}
