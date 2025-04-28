package repository

import "sync"

type Repository interface {
	Save(date string, rate float64)
	Get(date string) (float64, bool)
	All() map[string]float64
}

type InMemory struct {
	sync.RWMutex
	Rates map[string]float64
}

func NewInMemory() *InMemory {
	return &InMemory{
		Rates: make(map[string]float64),
	}
}

func (im *InMemory) Save(date string, rate float64) {
	im.Lock()
	defer im.Unlock()
	im.Rates[date] = rate
}

func (im *InMemory) Get(date string) (float64, bool) {
	rate, exist := im.Rates[date]
	return rate, exist
}

func (im *InMemory) All() map[string]float64 {
	return im.Rates
}
