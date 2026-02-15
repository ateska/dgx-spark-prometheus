package collectors

import (
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// GPUCollector collects GPU metrics via nvidia-smi.
type GPUCollector struct {
	utilizationDesc *prometheus.Desc
	tempDesc        *prometheus.Desc
	freqDesc        *prometheus.Desc
	powerDesc       *prometheus.Desc
}

// NewGPUCollector creates a new GPUCollector.
func NewGPUCollector() *GPUCollector {
	return &GPUCollector{
		utilizationDesc: prometheus.NewDesc(
			"gpu_utilization_percent",
			"GPU (GB10) utilization percentage (0-100)",
			nil, nil,
		),
		tempDesc: prometheus.NewDesc(
			"gpu_temperature_celsius",
			"GPU temperature in degrees Celsius",
			nil, nil,
		),
		freqDesc: prometheus.NewDesc(
			"gpu_frequency_mhz",
			"GPU graphics clock frequency in MHz",
			nil, nil,
		),
		powerDesc: prometheus.NewDesc(
			"gpu_power_watts",
			"GPU power consumption in Watts",
			nil, nil,
		),
	}
}

// Describe sends metric descriptors to the channel.
func (c *GPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.utilizationDesc
	ch <- c.tempDesc
	ch <- c.freqDesc
	ch <- c.powerDesc
}

// Collect runs nvidia-smi and sends GPU metrics to the channel.
// If nvidia-smi is not available or fails, no metrics are emitted.
func (c *GPUCollector) Collect(ch chan<- prometheus.Metric) {
	out, err := exec.Command(
		"nvidia-smi",
		"--query-gpu=utilization.gpu,temperature.gpu,power.draw,clocks.current.graphics",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		log.Printf("nvidia-smi failed: %v", err)
		return
	}

	// Parse the first GPU line (DGX Spark has one GPU)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return
	}

	fields := strings.Split(lines[0], ",")
	if len(fields) < 4 {
		log.Printf("nvidia-smi: unexpected output format: %q", lines[0])
		return
	}

	utilization := parseNvidiaSmiFloat(fields[0])
	temp := parseNvidiaSmiFloat(fields[1])
	power := parseNvidiaSmiFloat(fields[2])
	freq := parseNvidiaSmiFloat(fields[3])

	ch <- prometheus.MustNewConstMetric(c.utilizationDesc, prometheus.GaugeValue, utilization)
	ch <- prometheus.MustNewConstMetric(c.tempDesc, prometheus.GaugeValue, temp)
	ch <- prometheus.MustNewConstMetric(c.freqDesc, prometheus.GaugeValue, freq)
	ch <- prometheus.MustNewConstMetric(c.powerDesc, prometheus.GaugeValue, power)
}

// parseNvidiaSmiFloat parses a float from nvidia-smi output, handling N/A values.
func parseNvidiaSmiFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "[N/A]" || s == "N/A" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
