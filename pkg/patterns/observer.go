package patterns

import "sync"

type Observer interface {
	Notify(event interface{})
}

type Observable interface {
	AddObserver(Observer)
	RemoveObserver(Observer)
	NotifyAll(interface{})
}

type ObserverSet map[Observer]struct{}

var member = struct{}{}

type SimpleObservable struct {
	sync.RWMutex
	Observers ObserverSet
}

func NewSimpleObservable() *SimpleObservable {
	return &SimpleObservable{
		RWMutex:   sync.RWMutex{},
		Observers: make(ObserverSet),
	}
}

func (o *SimpleObservable) AddObserver(observer Observer) {
	o.Lock()
	o.Observers[observer] = member
	o.Unlock()
}

func (o *SimpleObservable) RemoveObserver(observer Observer) {
	o.Lock()
	delete(o.Observers, observer)
	o.Unlock()
}

func (o *SimpleObservable) NotifyAll(event interface{}) {
	o.RLock()
	defer o.RUnlock()
	for observer, _ := range o.Observers {
		observer.Notify(event)
	}
}
