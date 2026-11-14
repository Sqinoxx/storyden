package server_metrics

import (
	"sync/atomic"
	"time"
)

var latencyBucketsMs = []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}

type Requests struct {
	count     atomic.Uint64
	errors    atomic.Uint64
	sumMicros atomic.Uint64
	buckets   [12]atomic.Uint64
}

func NewRequests() *Requests {
	return &Requests{}
}

func (r *Requests) Observe(d time.Duration, status int) {
	r.count.Add(1)
	if status >= 500 {
		r.errors.Add(1)
	}
	r.sumMicros.Add(uint64(d.Microseconds()))

	ms := float64(d.Microseconds()) / 1000
	i := 0
	for i < len(latencyBucketsMs) && ms > latencyBucketsMs[i] {
		i++
	}
	r.buckets[i].Add(1)
}

type RequestWindow struct {
	Count  uint64
	Errors uint64
	AvgMs  float64
	P95Ms  float64
}

func (r *Requests) Drain() RequestWindow {
	count := r.count.Swap(0)
	errors := r.errors.Swap(0)
	sum := r.sumMicros.Swap(0)

	var buckets [12]uint64
	var total uint64
	for i := range r.buckets {
		buckets[i] = r.buckets[i].Swap(0)
		total += buckets[i]
	}

	w := RequestWindow{Count: count, Errors: errors}
	if count > 0 {
		w.AvgMs = float64(sum) / float64(count) / 1000
	}
	w.P95Ms = percentile(buckets[:], total, 0.95)

	return w
}

func percentile(buckets []uint64, total uint64, q float64) float64 {
	if total == 0 {
		return 0
	}

	target := uint64(float64(total)*q + 0.999999)
	var seen uint64
	for i, n := range buckets {
		seen += n
		if seen >= target {
			if i < len(latencyBucketsMs) {
				return latencyBucketsMs[i]
			}
			return latencyBucketsMs[len(latencyBucketsMs)-1] * 2
		}
	}

	return 0
}
