# DGX Spark Prometheus Exporter

A Prometheus metrics exporter for NVIDIA DGX Spark systems. It exposes hardware and system metrics via an HTTP endpoint that Prometheus can scrape.

## Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `cpu_usage_percent` | Gauge | CPU usage percentage (0-100) |
| `cpu_temperature_celsius` | Gauge | CPU temperature in °C |
| `cpu_frequency_mhz` | Gauge | Average CPU core frequency in MHz |
| `gpu_utilization_percent` | Gauge | GPU (GB10) utilization percentage |
| `gpu_temperature_celsius` | Gauge | GPU temperature in °C |
| `gpu_frequency_mhz` | Gauge | GPU graphics clock in MHz |
| `gpu_power_watts` | Gauge | GPU power consumption in Watts |
| `memory_total_bytes` | Gauge | Total RAM in bytes |
| `memory_used_bytes` | Gauge | Used RAM in bytes |
| `diskio_reads_completed_total` | Counter | Disk read operations (label: `device`) |
| `diskio_writes_completed_total` | Counter | Disk write operations (label: `device`) |
| `storage_used_percent` | Gauge | Used capacity of `/` in percent |
| `network_receive_bytes_total` | Counter | Bytes received (label: `interface`) |
| `network_transmit_bytes_total` | Counter | Bytes transmitted (label: `interface`) |
| `network_receive_packets_total` | Counter | Packets received (label: `interface`) |
| `network_transmit_packets_total` | Counter | Packets transmitted (label: `interface`) |

### Deriving Rates in PromQL

For counter metrics, use `rate()` or `irate()` in Prometheus to compute per-second rates:

```promql
# Disk IOPS (read)
rate(disk_reads_completed_total{device="nvme0n1"}[5m])

# Network throughput in bytes/sec
rate(network_receive_bytes_total{interface="enP7s7"}[5m])
```

## Monitored Network Interfaces

Only the following interfaces are monitored (when they are up):

- `enP7s7`
- `enp1s0f1np1`
- `enP2p1s0f1np1`
- `enp1s0f0np0`
- `enP2p1s0f0np0`
- `wlP9s9`

## Building and installing

Run on the DGX Spark, with the Go language installad, in the root directory of this repository:

```bash
go build .
```

```bash
sudo cp ./dgx-spark-prometheus /usr/local/bin/
sudo cp ./dgx-spark-prometheus.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now dgx-spark-prometheus
sudo systemctl start dgx-spark-prometheus
```

### How to transfer build to another DGX Spark

On the originating DGX Spark `spark1`:

```
tladmin@spark1:~/dgx-spark-prometheus$ scp dgx-spark-prometheus dgx-spark-prometheus.service spark2:~
```

On the receiveing DGX Spark `spark2`:

```
tladmin@spark2:~ sudo mv ./dgx-spark-prometheus /usr/local/bin/
tladmin@spark2:~ sudo mv ./dgx-spark-prometheus.service /etc/systemd/system/
tladmin@spark2:~ sudo systemctl daemon-reload
tladmin@spark2:~ sudo systemctl enable --now dgx-spark-prometheus
tladmin@spark2:~ sudo systemctl start dgx-spark-prometheus
```

## Prometheus configuration

```
scrape_configs:
  - job_name: 'dgx_spark'
    scrape_interval: 5s
    static_configs:
      - targets: ['spark1:9835', 'spark2:9835', ...]
    metrics_path: /metrics
    scheme: http
```


## Data Sources

| Metric | Source |
|--------|--------|
| CPU usage | `/proc/stat` (delta between scrapes) |
| CPU temperature | `/sys/class/thermal/thermal_zone*/` |
| CPU frequency | `/sys/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq` |
| GPU metrics | `nvidia-smi --query-gpu=...` |
| Memory | `/proc/meminfo` |
| Disk I/O | `/proc/diskstats` |
| Disk capacity | `statfs("/")` |
| Network I/O | `/sys/class/net/<iface>/statistics/` |
