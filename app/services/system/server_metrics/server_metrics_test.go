package server_metrics

import (
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/sensors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestsDrain(t *testing.T) {
	a := assert.New(t)
	r := NewRequests()

	for range 94 {
		r.Observe(8*time.Millisecond, 200)
	}
	for range 5 {
		r.Observe(300*time.Millisecond, 200)
	}
	r.Observe(20*time.Second, 503)

	w := r.Drain()
	a.Equal(uint64(100), w.Count)
	a.Equal(uint64(1), w.Errors)
	a.Equal(float64(500), w.P95Ms)
	a.InDelta((94*8+5*300+20000)/100.0, w.AvgMs, 0.01)

	empty := r.Drain()
	a.Zero(empty.Count)
	a.Zero(empty.P95Ms)
	a.Zero(empty.AvgMs)
}

func TestPercentileOverflowBucket(t *testing.T) {
	r := NewRequests()
	r.Observe(time.Minute, 200)
	assert.Equal(t, float64(20000), r.Drain().P95Ms)
}

func TestHistoryRanges(t *testing.T) {
	r := require.New(t)
	h := NewHistory()

	start := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	total := fineCapacity + 30
	for i := range total {
		h.Add(Point{Time: start.Add(time.Duration(i) * SampleInterval), CPU: float64(i % 12), P95Ms: float64(i)})
	}

	hour := h.Get(RangeHour)
	r.Len(hour, fineCapacity)
	r.True(hour[0].Time.Before(hour[len(hour)-1].Time))
	r.Equal(start.Add(time.Duration(total-1)*SampleInterval), hour[len(hour)-1].Time)

	live := h.Get(RangeLive)
	r.Len(live, liveWindow)
	r.Equal(hour[len(hour)-1], live[len(live)-1])

	day := h.Get(RangeDay)
	completeMinutes := total / 12
	r.Len(day, completeMinutes+1)
	r.Equal(start, day[0].Time)
	r.Equal(5.5, day[0].CPU)
	r.Equal(float64(11), day[0].P95Ms)
	r.Equal(start.Add(time.Minute), day[1].Time)
}

func TestCoarseRingWraps(t *testing.T) {
	h := NewHistory()
	start := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	for i := range coarseCapacity + 10 {
		h.Add(Point{Time: start.Add(time.Duration(i) * time.Minute)})
	}

	day := h.Get(RangeDay)
	require.Len(t, day, coarseCapacity+1)
	assert.Equal(t, start.Add(10*time.Minute-time.Minute), day[0].Time)
}

func TestBusyPercent(t *testing.T) {
	a := assert.New(t)
	prev := cpu.TimesStat{User: 100, System: 50, Idle: 800, Iowait: 50}
	cur := cpu.TimesStat{User: 130, System: 60, Idle: 850, Iowait: 60}

	a.InDelta(40.0, busyPercent(prev, cur), 0.001)
	a.Zero(busyPercent(cur, cur))
	a.Zero(busyPercent(cur, prev))
}

func TestSumDiskIOSkipsPartitionsAndVirtual(t *testing.T) {
	read, write := sumDiskIO(map[string]disk.IOCountersStat{
		"sda":       {ReadBytes: 100, WriteBytes: 10},
		"sda1":      {ReadBytes: 60, WriteBytes: 5},
		"nvme0n1":   {ReadBytes: 200, WriteBytes: 20},
		"nvme0n1p2": {ReadBytes: 200, WriteBytes: 20},
		"dm-0":      {ReadBytes: 999, WriteBytes: 999},
		"loop3":     {ReadBytes: 999, WriteBytes: 999},
	})
	assert.Equal(t, uint64(300), read)
	assert.Equal(t, uint64(30), write)
}

func TestSumNetIOSkipsLoopback(t *testing.T) {
	recv, sent := sumNetIO([]net.IOCountersStat{
		{Name: "lo", BytesRecv: 1000, BytesSent: 1000},
		{Name: "eth0", BytesRecv: 5, BytesSent: 7},
		{Name: "eth1", BytesRecv: 1, BytesSent: 1},
	})
	assert.Equal(t, uint64(6), recv)
	assert.Equal(t, uint64(8), sent)
}

func TestCPUTemperature(t *testing.T) {
	temp, ok := cpuTemperature([]sensors.TemperatureStat{
		{SensorKey: "nvme_composite", Temperature: 60},
		{SensorKey: "coretemp_package_id_0", Temperature: 48},
		{SensorKey: "coretemp_core_0", Temperature: 52},
		{SensorKey: "bogus", Temperature: 255},
	})
	assert.True(t, ok)
	assert.Equal(t, 52.0, temp)

	temp, ok = cpuTemperature([]sensors.TemperatureStat{{SensorKey: "acpitz", Temperature: 41}})
	assert.True(t, ok)
	assert.Equal(t, 41.0, temp)

	_, ok = cpuTemperature(nil)
	assert.False(t, ok)
}

func TestRate(t *testing.T) {
	assert.Equal(t, 20.0, rate(100, 200, 5))
	assert.Zero(t, rate(200, 100, 5))
	assert.Zero(t, rate(100, 200, 0))
}

func TestOllamaURL(t *testing.T) {
	cases := map[string]string{
		"http://ollama:11434/v1":    "http://ollama:11434/api/ps",
		"http://ollama:11434/v1/":   "http://ollama:11434/api/ps",
		"https://api.openai.com/v1": "",
		"":                          "",
	}
	for base, want := range cases {
		s := &Service{}
		s.cfg.LanguageModelProvider = "openai"
		s.cfg.OpenAIBaseURL = base
		assert.Equal(t, want, s.ollamaURL(), base)
	}
}
