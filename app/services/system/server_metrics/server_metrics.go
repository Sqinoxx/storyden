package server_metrics

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/shirou/gopsutil/v4/sensors"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/internal/config"
)

const slowEvery = 6

type Static struct {
	CPUModel      string
	Cores         int
	OS            string
	KernelVersion string
	AppStartedAt  time.Time
}

type Current struct {
	Time            time.Time
	CPUPercent      float64
	CPUPerCore      []float64
	Load1           float64
	Load5           float64
	Load15          float64
	MemTotal        uint64
	MemUsed         uint64
	MemAvailable    uint64
	SwapTotal       uint64
	SwapUsed        uint64
	DiskTotal       uint64
	DiskUsed        uint64
	DiskReadBps     float64
	DiskWriteBps    float64
	NetInBps        float64
	NetOutBps       float64
	RequestsPerSec  float64
	AvgLatencyMs    float64
	P95LatencyMs    float64
	Errors          uint64
	Goroutines      int
	HeapBytes       uint64
	ProcessCPU      float64
	ProcessRSS      uint64
	HostUptime      uint64
	TemperatureC    *float64
	DBOpen          int
	DBInUse         int
	DBActive        *int
	DBTotal         *int
	DBSizeBytes     *int64
	OCRQueue        *int
	AIModels        []AIModel
	AIEnabled       bool
}

type AIModel struct {
	Name      string
	SizeBytes int64
}

type Snapshot struct {
	Static  Static
	Current Current
	History []Point
}

type Service struct {
	logger   *slog.Logger
	cfg      config.Config
	db       *sql.DB
	assets   *asset_querier.Querier
	requests *Requests
	history  *History
	client   *http.Client

	mu      sync.RWMutex
	static  Static
	current Current

	proc      *process.Process
	diskPath  string
	prevTime  time.Time
	prevCPU   *cpu.TimesStat
	prevCores []cpu.TimesStat
	prevProc  *cpu.TimesStat
	prevRead  uint64
	prevWrite uint64
	prevRecv  uint64
	prevSent  uint64
	ticks     int
}

func Build() fx.Option {
	return fx.Options(
		fx.Provide(NewRequests),
		fx.Provide(New),
		fx.Invoke(func(*Service) {}),
	)
}

func New(
	ctx context.Context,
	lc fx.Lifecycle,
	logger *slog.Logger,
	cfg config.Config,
	db *sql.DB,
	assets *asset_querier.Querier,
	requests *Requests,
) *Service {
	s := &Service{
		logger:   logger,
		cfg:      cfg,
		db:       db,
		assets:   assets,
		requests: requests,
		history:  NewHistory(),
		client:   &http.Client{Timeout: 2 * time.Second},
		diskPath: cfg.AssetStorageLocalPath,
		static:   Static{AppStartedAt: time.Now()},
	}
	if s.diskPath == "" {
		s.diskPath = "./data"
	}

	jobCtx, cancel := context.WithCancel(ctx)

	lc.Append(fx.StartHook(func() error {
		go s.run(jobCtx)
		return nil
	}))
	lc.Append(fx.StopHook(func() error {
		cancel()
		return nil
	}))

	return s
}

func (s *Service) Get(r Range) Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cur := s.current
	cur.CPUPerCore = append([]float64(nil), s.current.CPUPerCore...)
	cur.AIModels = append([]AIModel(nil), s.current.AIModels...)

	return Snapshot{
		Static:  s.static,
		Current: cur,
		History: s.history.Get(r),
	}
}

func (s *Service) run(ctx context.Context) {
	s.loadStatic()
	s.sample(ctx, time.Now())

	ticker := time.NewTicker(SampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.sample(ctx, now)
		}
	}
}

func (s *Service) loadStatic() {
	st := Static{AppStartedAt: s.static.AppStartedAt}

	if infos, err := cpu.Info(); err == nil && len(infos) > 0 {
		st.CPUModel = strings.TrimSpace(infos[0].ModelName)
	}
	if n, err := cpu.Counts(true); err == nil {
		st.Cores = n
	}
	if info, err := host.Info(); err == nil {
		st.OS = strings.TrimSpace(info.Platform + " " + info.PlatformVersion)
		st.KernelVersion = info.KernelVersion
	}
	if p, err := process.NewProcess(int32(os.Getpid())); err == nil {
		s.proc = p
	}

	s.mu.Lock()
	s.static = st
	s.mu.Unlock()
}

func (s *Service) sample(ctx context.Context, now time.Time) {
	elapsed := now.Sub(s.prevTime).Seconds()
	first := s.prevTime.IsZero()
	s.prevTime = now

	s.mu.RLock()
	c := s.current
	s.mu.RUnlock()
	c.Time = now

	if times, err := cpu.Times(false); err == nil && len(times) > 0 {
		if s.prevCPU != nil {
			c.CPUPercent = busyPercent(*s.prevCPU, times[0])
		}
		s.prevCPU = &times[0]
	}

	if cores, err := cpu.Times(true); err == nil {
		if len(s.prevCores) == len(cores) {
			perCore := make([]float64, len(cores))
			for i := range cores {
				perCore[i] = busyPercent(s.prevCores[i], cores[i])
			}
			c.CPUPerCore = perCore
		}
		s.prevCores = cores
	}

	if l, err := load.Avg(); err == nil {
		c.Load1, c.Load5, c.Load15 = l.Load1, l.Load5, l.Load15
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		c.MemTotal = vm.Total
		c.MemAvailable = vm.Available
		c.MemUsed = vm.Total - vm.Available
	}
	if sw, err := mem.SwapMemory(); err == nil {
		c.SwapTotal, c.SwapUsed = sw.Total, sw.Used
	}

	if u, err := disk.Usage(s.diskPath); err == nil {
		c.DiskTotal, c.DiskUsed = u.Total, u.Used
	}

	if counters, err := disk.IOCounters(); err == nil {
		read, write := sumDiskIO(counters)
		if !first {
			c.DiskReadBps = rate(s.prevRead, read, elapsed)
			c.DiskWriteBps = rate(s.prevWrite, write, elapsed)
		}
		s.prevRead, s.prevWrite = read, write
	}

	if counters, err := net.IOCounters(true); err == nil {
		recv, sent := sumNetIO(counters)
		if !first {
			c.NetInBps = rate(s.prevRecv, recv, elapsed)
			c.NetOutBps = rate(s.prevSent, sent, elapsed)
		}
		s.prevRecv, s.prevSent = recv, sent
	}

	window := s.requests.Drain()
	if !first && elapsed > 0 {
		c.RequestsPerSec = float64(window.Count) / elapsed
	}
	c.AvgLatencyMs = window.AvgMs
	c.P95LatencyMs = window.P95Ms
	c.Errors = window.Errors

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	c.Goroutines = runtime.NumGoroutine()
	c.HeapBytes = ms.HeapAlloc

	if s.proc != nil {
		if t, err := s.proc.Times(); err == nil {
			if s.prevProc != nil && elapsed > 0 {
				cores := max(1, s.static.Cores)
				used := (t.User + t.System) - (s.prevProc.User + s.prevProc.System)
				c.ProcessCPU = min(100, max(0, used/elapsed/float64(cores)*100))
			}
			s.prevProc = t
		}
		if m, err := s.proc.MemoryInfo(); err == nil {
			c.ProcessRSS = m.RSS
		}
	}

	if up, err := host.Uptime(); err == nil {
		c.HostUptime = up
	}

	c.TemperatureC = nil
	if temps, _ := sensors.SensorsTemperatures(); len(temps) > 0 {
		if t, ok := cpuTemperature(temps); ok {
			c.TemperatureC = &t
		}
	}

	stats := s.db.Stats()
	c.DBOpen, c.DBInUse = stats.OpenConnections, stats.InUse

	if s.ticks%slowEvery == 0 {
		s.sampleSlow(ctx, &c)
	}
	s.ticks++

	s.mu.Lock()
	s.current = c
	s.mu.Unlock()

	if !first {
		s.history.Add(Point{
			Time:      now,
			CPU:       c.CPUPercent,
			Mem:       percentOf(c.MemUsed, c.MemTotal),
			NetIn:     c.NetInBps,
			NetOut:    c.NetOutBps,
			DiskRead:  c.DiskReadBps,
			DiskWrite: c.DiskWriteBps,
			RPS:       c.RequestsPerSec,
			P95Ms:     c.P95LatencyMs,
			Errors:    float64(c.Errors),
		})
	}
}

func (s *Service) sampleSlow(ctx context.Context, c *Current) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if s.isPostgres() {
		var active, total int
		var size int64
		err := s.db.QueryRowContext(ctx, `
			SELECT
				count(*) FILTER (WHERE state = 'active' AND pid <> pg_backend_pid()),
				count(*),
				pg_database_size(current_database())
			FROM pg_stat_activity
			WHERE datname = current_database()`).Scan(&active, &total, &size)
		if err == nil {
			c.DBActive, c.DBTotal, c.DBSizeBytes = &active, &total, &size
		}
	} else {
		var size int64
		err := s.db.QueryRowContext(ctx, `SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()`).Scan(&size)
		if err == nil {
			c.DBSizeBytes = &size
		}
	}

	if stats, err := s.assets.GetOCRStats(ctx); err == nil {
		queue := stats.Pending + stats.Processing
		c.OCRQueue = &queue
	}

	c.AIModels, c.AIEnabled = s.ollamaModels(ctx)
}

func (s *Service) isPostgres() bool {
	u, err := url.Parse(s.cfg.DatabaseURL)
	if err != nil {
		return false
	}
	return u.Scheme == "postgres" || u.Scheme == "postgresql"
}

func (s *Service) ollamaURL() string {
	base := strings.TrimRight(s.cfg.OpenAIBaseURL, "/")
	if s.cfg.LanguageModelProvider == "" || base == "" || strings.Contains(base, "openai.com") {
		return ""
	}
	return strings.TrimSuffix(base, "/v1") + "/api/ps"
}

func (s *Service) ollamaModels(ctx context.Context) ([]AIModel, bool) {
	endpoint := s.ollamaURL()
	if endpoint == "" {
		return nil, false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false
	}

	var body struct {
		Models []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, false
	}

	models := make([]AIModel, 0, len(body.Models))
	for _, m := range body.Models {
		models = append(models, AIModel{Name: m.Name, SizeBytes: m.Size})
	}

	return models, true
}
