package collectors

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
)

// DiskCollector collects disk I/O counters and root filesystem capacity.
type DiskCollector struct {
	readsDesc  *prometheus.Desc
	writesDesc *prometheus.Desc
	usedDesc   *prometheus.Desc
}

// NewDiskCollector creates a new DiskCollector.
func NewDiskCollector() *DiskCollector {
	return &DiskCollector{
		readsDesc: prometheus.NewDesc(
			"diskio_reads_completed_total",
			"Total number of completed disk read operations (use rate() in PromQL for IOPS)",
			[]string{"device"}, nil,
		),
		writesDesc: prometheus.NewDesc(
			"diskio_writes_completed_total",
			"Total number of completed disk write operations (use rate() in PromQL for IOPS)",
			[]string{"device"}, nil,
		),
		usedDesc: prometheus.NewDesc(
			"storage_used_percent",
			"Used storage capacity of / filesystem in percent",
			nil, nil,
		),
	}
}

// Describe sends metric descriptors to the channel.
func (c *DiskCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.readsDesc
	ch <- c.writesDesc
	ch <- c.usedDesc
}

// Collect reads disk I/O stats and root capacity, sending them to the channel.
func (c *DiskCollector) Collect(ch chan<- prometheus.Metric) {
	c.collectDiskIO(ch)
	c.collectRootCapacity(ch)
}

// collectDiskIO reads /proc/diskstats for physical disk devices.
func (c *DiskCollector) collectDiskIO(ch chan<- prometheus.Metric) {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return
	}
	defer f.Close()

	physicalPrefixes := []string{"sd", "nvme", "vd", "hd", "xvd", "mmcblk"}
	excludePrefixes := []string{"loop", "ram", "dm-", "sr", "fd"}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}

		device := fields[2]

		// Skip excluded devices
		if hasAnyPrefix(device, excludePrefixes) {
			continue
		}

		// Only include physical devices
		if !hasAnyPrefix(device, physicalPrefixes) {
			continue
		}

		// Field 3: reads completed, Field 7: writes completed
		// See https://www.kernel.org/doc/Documentation/ABI/testing/procfs-diskstats
		reads, _ := strconv.ParseFloat(fields[3], 64)
		writes, _ := strconv.ParseFloat(fields[7], 64)

		ch <- prometheus.MustNewConstMetric(c.readsDesc, prometheus.CounterValue, reads, device)
		ch <- prometheus.MustNewConstMetric(c.writesDesc, prometheus.CounterValue, writes, device)
	}
}

// collectRootCapacity reports the used capacity percentage of the / filesystem.
func (c *DiskCollector) collectRootCapacity(ch chan<- prometheus.Metric) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return
	}

	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)

	if total == 0 {
		return
	}

	// Used percentage from the user's perspective (total - available) / total
	usedPercent := float64(total-available) / float64(total) * 100.0
	ch <- prometheus.MustNewConstMetric(c.usedDesc, prometheus.GaugeValue, usedPercent)
}

// hasAnyPrefix checks if s starts with any of the given prefixes.
func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
