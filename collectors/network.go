package collectors

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// monitoredInterfaces is the fixed list of network interfaces to monitor on DGX Spark.
// Only interfaces that are currently "up" will have metrics emitted.
var monitoredInterfaces = []string{
	"enP7s7",
	"enp1s0f1np1",
	"enP2p1s0f1np1",
	"enp1s0f0np0",
	"enP2p1s0f0np0",
	"wlP9s9",
}

// NetworkCollector collects per-interface network I/O counters.
type NetworkCollector struct {
	rxBytesDesc   *prometheus.Desc
	txBytesDesc   *prometheus.Desc
	rxPacketsDesc *prometheus.Desc
	txPacketsDesc *prometheus.Desc
}

// NewNetworkCollector creates a new NetworkCollector.
func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{
		rxBytesDesc: prometheus.NewDesc(
			"network_receive_bytes_total",
			"Total bytes received on network interface",
			[]string{"interface"}, nil,
		),
		txBytesDesc: prometheus.NewDesc(
			"network_transmit_bytes_total",
			"Total bytes transmitted on network interface",
			[]string{"interface"}, nil,
		),
		rxPacketsDesc: prometheus.NewDesc(
			"network_receive_packets_total",
			"Total packets received on network interface",
			[]string{"interface"}, nil,
		),
		txPacketsDesc: prometheus.NewDesc(
			"network_transmit_packets_total",
			"Total packets transmitted on network interface",
			[]string{"interface"}, nil,
		),
	}
}

// Describe sends metric descriptors to the channel.
func (c *NetworkCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.rxBytesDesc
	ch <- c.txBytesDesc
	ch <- c.rxPacketsDesc
	ch <- c.txPacketsDesc
}

// Collect reads network interface statistics for monitored interfaces that are up.
func (c *NetworkCollector) Collect(ch chan<- prometheus.Metric) {
	for _, iface := range monitoredInterfaces {
		if !isInterfaceUp(iface) {
			continue
		}

		statsDir := filepath.Join("/sys/class/net", iface, "statistics")

		rxBytes := readSysUint64(filepath.Join(statsDir, "rx_bytes"))
		txBytes := readSysUint64(filepath.Join(statsDir, "tx_bytes"))
		rxPackets := readSysUint64(filepath.Join(statsDir, "rx_packets"))
		txPackets := readSysUint64(filepath.Join(statsDir, "tx_packets"))

		ch <- prometheus.MustNewConstMetric(c.rxBytesDesc, prometheus.CounterValue, float64(rxBytes), iface)
		ch <- prometheus.MustNewConstMetric(c.txBytesDesc, prometheus.CounterValue, float64(txBytes), iface)
		ch <- prometheus.MustNewConstMetric(c.rxPacketsDesc, prometheus.CounterValue, float64(rxPackets), iface)
		ch <- prometheus.MustNewConstMetric(c.txPacketsDesc, prometheus.CounterValue, float64(txPackets), iface)
	}
}

// isInterfaceUp checks if a network interface exists and has operstate "up".
func isInterfaceUp(iface string) bool {
	path := filepath.Join("/sys/class/net", iface, "operstate")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	state := strings.TrimSpace(string(data))
	return state == "up"
}

// readSysUint64 reads a sysfs file containing a single uint64 value.
func readSysUint64(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
