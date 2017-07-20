package main

import "sync"

var onceFuse sync.Once
var fuse     *Fuse

type Fuse struct {
    Monitors []Monitor
}

type Monitor interface {
    StartChecking(notifer *Notifer)
}

func GetFuse() *Fuse {
    onceFuse.Do(func(){
        fuse = &Fuse{
            Monitors: make([]Monitor, 0),
        }
    })
    return fuse
}

func (f *Fuse) AddMonitor(monitor Monitor){
    f.Monitors = append(f.Monitors, monitor)
}

func (f *Fuse) StartChecking(notifer *Notifer) {
    var wg sync.WaitGroup

    wg.Add(len(f.Monitors))
    for _, monitor := range f.Monitors {
        go func() {
            monitor.StartChecking(notifer)
            defer wg.Done()
        }()
    }
    wg.Wait()
}
