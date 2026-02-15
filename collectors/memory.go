package collectors

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// MemoryCollector collects RAM total and used metrics from /proc/meminfo.
type MemoryCollector struct {
	totalDesc *prometheus.Desc
	usedDesc  *prometheus.Desc
}

// NewMemoryCollector creates a new MemoryCollector.
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{
		totalDesc: prometheus.NewDesc(
			"memory_total_bytes",
			"Total physical RAM in bytes",
			nil, nil,
		),
		usedDesc: prometheus.NewDesc(
			"memory_used_bytes",
			"Used RAM in bytes (total - free - buffers - cached)",
			nil, nil,
		),
	}
}

// Describe sends metric descriptors to the channel.
func (c *MemoryCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalDesc
	ch <- c.usedDesc
}

// Collect reads /proc/meminfo and sends memory metrics to the channel.
func (c *MemoryCollector) Collect(ch chan<- prometheus.Metric) {
	memInfo, err := readMemInfo()
	if err != nil {
		return
	}

	totalKB := memInfo["MemTotal"]
	freeKB := memInfo["MemFree"]
	buffersKB := memInfo["Buffers"]
	cachedKB := memInfo["Cached"]

	totalBytes := float64(totalKB) * 1024
	usedBytes := float64(totalKB-freeKB-buffersKB-cachedKB) * 1024

	// Ensure used is non-negative (fallback: total - free)
	if usedBytes < 0 {
		usedBytes = float64(totalKB-freeKB) * 1024
	}

	ch <- prometheus.MustNewConstMetric(c.totalDesc, prometheus.GaugeValue, totalBytes)
	ch <- prometheus.MustNewConstMetric(c.usedDesc, prometheus.GaugeValue, usedBytes)
}

// readMemInfo parses /proc/meminfo into a map of key -> value in kB.
func readMemInfo() (map[string]uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info := make(map[string]uint64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		valStr = strings.TrimSuffix(valStr, " kB")
		valStr = strings.TrimSpace(valStr)

		val, err := strconv.ParseUint(valStr, 10, 64)
		if err != nil {
			continue
		}
		info[key] = val
	}

	return info, scanner.Err()
}
