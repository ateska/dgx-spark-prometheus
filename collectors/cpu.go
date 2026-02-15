package collectors

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// CPUCollector collects CPU usage, temperature, and frequency metrics.
type CPUCollector struct {
	usageDesc *prometheus.Desc
	tempDesc  *prometheus.Desc
	freqDesc  *prometheus.Desc

	mu        sync.Mutex
	prevIdle  uint64
	prevTotal uint64
}

// NewCPUCollector creates a new CPUCollector.
func NewCPUCollector() *CPUCollector {
	return &CPUCollector{
		usageDesc: prometheus.NewDesc(
			"cpu_usage_percent",
			"CPU usage percentage (0-100)",
			nil, nil,
		),
		tempDesc: prometheus.NewDesc(
			"cpu_temperature_celsius",
			"CPU temperature in degrees Celsius",
			nil, nil,
		),
		freqDesc: prometheus.NewDesc(
			"cpu_frequency_mhz",
			"Average CPU core frequency in MHz",
			nil, nil,
		),
	}
}

// Describe sends metric descriptors to the channel.
func (c *CPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usageDesc
	ch <- c.tempDesc
	ch <- c.freqDesc
}

// Collect reads current CPU metrics and sends them to the channel.
func (c *CPUCollector) Collect(ch chan<- prometheus.Metric) {
	if usage, ok := c.readCPUUsage(); ok {
		ch <- prometheus.MustNewConstMetric(c.usageDesc, prometheus.GaugeValue, usage)
	}

	if temp, ok := readCPUTemperature(); ok {
		ch <- prometheus.MustNewConstMetric(c.tempDesc, prometheus.GaugeValue, temp)
	}

	if freq, ok := readCPUFrequency(); ok {
		ch <- prometheus.MustNewConstMetric(c.freqDesc, prometheus.GaugeValue, freq)
	}
}

// readCPUUsage computes CPU usage percentage from /proc/stat deltas.
// The first scrape after startup returns 0 (no previous sample).
func (c *CPUCollector) readCPUUsage() (float64, bool) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			return 0, false
		}

		// Fields: cpu user nice system idle iowait irq softirq [steal guest guest_nice]
		user, _ := strconv.ParseUint(fields[1], 10, 64)
		nice, _ := strconv.ParseUint(fields[2], 10, 64)
		system, _ := strconv.ParseUint(fields[3], 10, 64)
		idle, _ := strconv.ParseUint(fields[4], 10, 64)
		iowait, _ := strconv.ParseUint(fields[5], 10, 64)
		irq, _ := strconv.ParseUint(fields[6], 10, 64)
		softirq, _ := strconv.ParseUint(fields[7], 10, 64)

		total := user + nice + system + idle + iowait + irq + softirq
		idleTotal := idle + iowait

		c.mu.Lock()
		prevTotal := c.prevTotal
		prevIdle := c.prevIdle
		c.prevTotal = total
		c.prevIdle = idleTotal
		c.mu.Unlock()

		// First sample: no delta available
		if prevTotal == 0 {
			return 0, true
		}

		totalDelta := total - prevTotal
		idleDelta := idleTotal - prevIdle

		if totalDelta == 0 {
			return 0, true
		}

		usage := float64(totalDelta-idleDelta) / float64(totalDelta) * 100.0
		return usage, true
	}

	return 0, false
}

// readCPUTemperature reads CPU temperature from thermal zones.
// It looks for a zone whose type contains "cpu" or "soc"; falls back to zone 0.
func readCPUTemperature() (float64, bool) {
	// Search for a CPU/SoC thermal zone
	for i := 0; i < 10; i++ {
		typePath := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/type", i)
		typeBytes, err := os.ReadFile(typePath)
		if err != nil {
			continue
		}

		zoneType := strings.ToLower(strings.TrimSpace(string(typeBytes)))
		if strings.Contains(zoneType, "cpu") || strings.Contains(zoneType, "soc") {
			tempPath := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", i)
			return readThermalTemp(tempPath)
		}
	}

	// Fallback: thermal_zone0
	return readThermalTemp("/sys/class/thermal/thermal_zone0/temp")
}

// readThermalTemp reads a thermal zone temp file (millidegrees) and returns Celsius.
func readThermalTemp(path string) (float64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	millideg, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return 0, false
	}
	return millideg / 1000.0, true
}

// readCPUFrequency returns the average CPU frequency in MHz across all cores.
// It reads scaling_cur_freq (in kHz) for each CPU core.
func readCPUFrequency() (float64, bool) {
	var totalFreq float64
	count := 0

	for i := 0; i < 256; i++ {
		path := fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/scaling_cur_freq", i)
		data, err := os.ReadFile(path)
		if err != nil {
			if i == 0 {
				// No cpufreq support at all
				return 0, false
			}
			break
		}
		freqKHz, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		if err != nil {
			continue
		}
		totalFreq += freqKHz
		count++
	}

	if count == 0 {
		return 0, false
	}

	// Convert kHz to MHz
	return totalFreq / float64(count) / 1000.0, true
}
