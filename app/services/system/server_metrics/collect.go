package server_metrics

import (
	"math"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/sensors"
)

func busyPercent(prev, cur cpu.TimesStat) float64 {
	prevAll := prev.Total() - prev.Guest - prev.GuestNice
	curAll := cur.Total() - cur.Guest - cur.GuestNice
	prevBusy := prevAll - prev.Idle - prev.Iowait
	curBusy := curAll - cur.Idle - cur.Iowait

	if curAll <= prevAll || curBusy <= prevBusy {
		return 0
	}

	return math.Min(100, (curBusy-prevBusy)/(curAll-prevAll)*100)
}

var virtualDiskPrefixes = []string{"loop", "ram", "zram", "dm-", "md", "sr", "fd", "nbd"}

func sumDiskIO(counters map[string]disk.IOCountersStat) (read, write uint64) {
	for name, c := range counters {
		if isVirtualDisk(name) || isPartition(name, counters) {
			continue
		}
		read += c.ReadBytes
		write += c.WriteBytes
	}
	return read, write
}

func isVirtualDisk(name string) bool {
	for _, p := range virtualDiskPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func isPartition(name string, counters map[string]disk.IOCountersStat) bool {
	for other := range counters {
		if other != name && strings.HasPrefix(name, other) {
			return true
		}
	}
	return false
}

func sumNetIO(counters []net.IOCountersStat) (recv, sent uint64) {
	for _, c := range counters {
		if c.Name == "lo" || strings.HasPrefix(c.Name, "Loopback") {
			continue
		}
		recv += c.BytesRecv
		sent += c.BytesSent
	}
	return recv, sent
}

var cpuSensorHints = []string{"coretemp", "k10temp", "zenpower", "cpu", "package", "tctl", "soc"}

func cpuTemperature(temps []sensors.TemperatureStat) (float64, bool) {
	valid := make([]sensors.TemperatureStat, 0, len(temps))
	for _, t := range temps {
		if t.Temperature > 0 && t.Temperature < 125 {
			valid = append(valid, t)
		}
	}
	if len(valid) == 0 {
		return 0, false
	}

	sort.Slice(valid, func(i, j int) bool { return valid[i].Temperature > valid[j].Temperature })

	for _, t := range valid {
		key := strings.ToLower(t.SensorKey)
		for _, hint := range cpuSensorHints {
			if strings.Contains(key, hint) {
				return t.Temperature, true
			}
		}
	}

	return valid[0].Temperature, true
}

func rate(prev, cur uint64, seconds float64) float64 {
	if cur < prev || seconds <= 0 {
		return 0
	}
	return float64(cur-prev) / seconds
}

func percentOf(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}
