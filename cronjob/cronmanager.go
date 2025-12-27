package cronjob

import "sync"

// CronManager holds all schedulers
type CronManager struct {
	mu         sync.Mutex
	schedulers []CronScheduler
}

func NewCronManager() *CronManager {
	return &CronManager{}
}

func (m *CronManager) AddScheduler(s CronScheduler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schedulers = append(m.schedulers, s)
}

func (m *CronManager) Shutdown(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.schedulers {
		s.Stop()
	}
}
