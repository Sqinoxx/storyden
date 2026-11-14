package server_metrics

import (
	"sync"
	"time"
)

const (
	SampleInterval = 5 * time.Second
	fineCapacity   = int(time.Hour / SampleInterval)
	coarseCapacity = 24 * 60
	liveWindow     = 60
)

type Point struct {
	Time      time.Time
	CPU       float64
	Mem       float64
	NetIn     float64
	NetOut    float64
	DiskRead  float64
	DiskWrite float64
	RPS       float64
	P95Ms     float64
	Errors    float64
}

type Range string

const (
	RangeLive Range = "live"
	RangeHour Range = "1h"
	RangeDay  Range = "24h"
)

type ring struct {
	buf  []Point
	next int
	full bool
}

func newRing(capacity int) ring {
	return ring{buf: make([]Point, capacity)}
}

func (r *ring) push(p Point) {
	r.buf[r.next] = p
	r.next = (r.next + 1) % len(r.buf)
	if r.next == 0 {
		r.full = true
	}
}

func (r *ring) all() []Point {
	if !r.full {
		return append([]Point(nil), r.buf[:r.next]...)
	}
	out := make([]Point, 0, len(r.buf))
	out = append(out, r.buf[r.next:]...)
	return append(out, r.buf[:r.next]...)
}

type History struct {
	mu      sync.RWMutex
	fine    ring
	coarse  ring
	pending []Point
}

func NewHistory() *History {
	return &History{
		fine:   newRing(fineCapacity),
		coarse: newRing(coarseCapacity),
	}
}

func (h *History) Add(p Point) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.fine.push(p)

	if len(h.pending) > 0 && !h.pending[0].Time.Truncate(time.Minute).Equal(p.Time.Truncate(time.Minute)) {
		h.coarse.push(aggregate(h.pending))
		h.pending = h.pending[:0]
	}
	h.pending = append(h.pending, p)
}

func (h *History) Get(r Range) []Point {
	h.mu.RLock()
	defer h.mu.RUnlock()

	switch r {
	case RangeDay:
		points := h.coarse.all()
		if len(h.pending) > 0 {
			points = append(points, aggregate(h.pending))
		}
		return points

	case RangeHour:
		return h.fine.all()

	default:
		points := h.fine.all()
		if len(points) > liveWindow {
			points = points[len(points)-liveWindow:]
		}
		return points
	}
}

func aggregate(points []Point) Point {
	out := Point{Time: points[0].Time.Truncate(time.Minute)}
	n := float64(len(points))
	for _, p := range points {
		out.CPU += p.CPU / n
		out.Mem += p.Mem / n
		out.NetIn += p.NetIn / n
		out.NetOut += p.NetOut / n
		out.DiskRead += p.DiskRead / n
		out.DiskWrite += p.DiskWrite / n
		out.RPS += p.RPS / n
		out.Errors += p.Errors
		out.P95Ms = max(out.P95Ms, p.P95Ms)
	}
	return out
}
