package collector

import (
	"sync"
	"time"
)

// cache durations
const (
	ModelCacheDuration       = 24 * time.Hour
	FrequencyCacheDuration   = 30 * time.Second
	TemperatureCacheDuration = 5 * time.Second
	UsageCacheDuration       = 1 * time.Second
)

// CPU Time Statistics
type CPUTimes struct {
	Total uint64
	Idle  uint64
}

// CPUCache holds cached CPU information
type CPUCache struct {
	// static info (rarely changes)
	model     string
	modelTime time.Time

	// Semi-static info (changes occasionally)
	frequency     float64
	frequencyTime time.Time

	// Temperature cache (changes frequently but can be cached briefly)
	temperature     float32
	temperatureTime time.Time

	// CPU usage previous readings
	previousStats    map[string]CPUTimes
	previousTime     time.Time
	cachedUsage      float64
	cachedCoreUsages []float64
	usageTime        time.Time

	mutex sync.RWMutex
}

// creating new cache struct
func NewCPUCache() *CPUCache {
	return &CPUCache{
		previousStats: make(map[string]CPUTimes),
	}
}

// checking if valid
func (c *CPUCache) IsModelCacheValid() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.model != "" && time.Since(c.modelTime) < ModelCacheDuration
}

func (c *CPUCache) IsFrequencyCacheValid() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.frequency != 0 && time.Since(c.frequencyTime) < FrequencyCacheDuration
}

func (c *CPUCache) IsTemperatureCacheValid() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.temperature != 0 && time.Since(c.temperatureTime) < TemperatureCacheDuration
}

func (c *CPUCache) IsUsageCacheValid() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.cachedUsage != 0 && time.Since(c.usageTime) < UsageCacheDuration
}

// getter and setters
func (c *CPUCache) GetCachedModel() (string, float64) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.model, c.frequency
}

func (c *CPUCache) SetCachedModel(model string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.model = model
	c.modelTime = time.Now()
}

func (c *CPUCache) GetCachedFrequency() float64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.frequency
}

func (c *CPUCache) SetCachedFrequency(frequency float64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.frequency = frequency
	c.frequencyTime = time.Now()
}

func (c *CPUCache) GetCachedTemperature() float32 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.temperature
}

func (c *CPUCache) SetCachedTemperature(temp float32) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.temperature = temp
	c.temperatureTime = time.Now()
}

func (c *CPUCache) GetCachedUsage() (float64, []float64) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.cachedUsage, c.cachedCoreUsages
}

func (c *CPUCache) SetCachedUsage(usage float64, coreUsages []float64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cachedUsage = usage
	c.cachedCoreUsages = coreUsages
	c.usageTime = time.Now()
}

func (c *CPUCache) GetPreviousStats() (map[string]CPUTimes, time.Time) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Return a copy to avoid race conditions
	statsCopy := make(map[string]CPUTimes)
	for k, v := range c.previousStats {
		statsCopy[k] = v
	}

	return statsCopy, c.previousTime
}

func (c *CPUCache) SetPreviousStats(stats map[string]CPUTimes) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.previousStats = stats
	c.previousTime = time.Now()
}

func (c *CPUCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.model = ""
	c.frequency = 0
	c.temperature = 0
	c.cachedUsage = 0
	c.cachedCoreUsages = nil
	c.previousStats = make(map[string]CPUTimes)
}

// GetTimeSinceLastUsageUpdate returns the time elapsed since last usage update
func (c *CPUCache) GetTimeSinceLastUsageUpdate() time.Duration {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return time.Since(c.previousTime)
}
