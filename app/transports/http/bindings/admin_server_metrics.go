package bindings

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"

	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/system/server_metrics"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

func (a *Admin) AdminServerMetrics(ctx context.Context, request openapi.AdminServerMetricsRequestObject) (openapi.AdminServerMetricsResponseObject, error) {
	if err := session.Authorise(ctx, nil, rbac.PermissionAdministrator); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	r := server_metrics.RangeLive
	if request.Params.Range != nil {
		r = server_metrics.Range(*request.Params.Range)
	}

	return openapi.AdminServerMetrics200JSONResponse(serialiseServerMetrics(a.serverMetrics.Get(r))), nil
}

func serialiseServerMetrics(s server_metrics.Snapshot) openapi.ServerMetrics {
	c := s.Current

	perCore := make([]float32, len(c.CPUPerCore))
	for i, v := range c.CPUPerCore {
		perCore[i] = float32(v)
	}

	models := make([]openapi.ServerMetricsAIModel, len(c.AIModels))
	for i, m := range c.AIModels {
		models[i] = openapi.ServerMetricsAIModel{Name: m.Name, SizeBytes: m.SizeBytes}
	}

	var temperature *float32
	if c.TemperatureC != nil {
		t := float32(*c.TemperatureC)
		temperature = &t
	}

	aiEnabled := c.AIEnabled

	history := make([]openapi.ServerMetricsPoint, len(s.History))
	for i, p := range s.History {
		history[i] = openapi.ServerMetricsPoint{
			T:         p.Time,
			Cpu:       float32(p.CPU),
			Mem:       float32(p.Mem),
			NetIn:     float32(p.NetIn),
			NetOut:    float32(p.NetOut),
			DiskRead:  float32(p.DiskRead),
			DiskWrite: float32(p.DiskWrite),
			Rps:       float32(p.RPS),
			P95Ms:     float32(p.P95Ms),
			Errors:    float32(p.Errors),
		}
	}

	return openapi.ServerMetrics{
		Static: openapi.ServerMetricsStatic{
			CpuModel:      s.Static.CPUModel,
			Cores:         s.Static.Cores,
			Os:            s.Static.OS,
			KernelVersion: s.Static.KernelVersion,
			AppStartedAt:  s.Static.AppStartedAt,
		},
		Current: openapi.ServerMetricsCurrent{
			Time:              c.Time,
			CpuPercent:        float32(c.CPUPercent),
			CpuPerCore:        perCore,
			Load1:             float32(c.Load1),
			Load5:             float32(c.Load5),
			Load15:            float32(c.Load15),
			MemTotal:          int64(c.MemTotal),
			MemUsed:           int64(c.MemUsed),
			MemAvailable:      int64(c.MemAvailable),
			SwapTotal:         int64(c.SwapTotal),
			SwapUsed:          int64(c.SwapUsed),
			DiskTotal:         int64(c.DiskTotal),
			DiskUsed:          int64(c.DiskUsed),
			DiskReadBps:       float32(c.DiskReadBps),
			DiskWriteBps:      float32(c.DiskWriteBps),
			NetInBps:          float32(c.NetInBps),
			NetOutBps:         float32(c.NetOutBps),
			RequestsPerSec:    float32(c.RequestsPerSec),
			AvgLatencyMs:      float32(c.AvgLatencyMs),
			P95LatencyMs:      float32(c.P95LatencyMs),
			Errors:            int(c.Errors),
			Goroutines:        c.Goroutines,
			HeapBytes:         int64(c.HeapBytes),
			ProcessCpuPercent: float32(c.ProcessCPU),
			ProcessRss:        int64(c.ProcessRSS),
			HostUptimeSeconds: int64(c.HostUptime),
			TemperatureC:      temperature,
			DbOpen:            c.DBOpen,
			DbInUse:           c.DBInUse,
			DbActive:          c.DBActive,
			DbTotal:           c.DBTotal,
			DbSizeBytes:       c.DBSizeBytes,
			OcrQueue:          c.OCRQueue,
			AiEnabled:         &aiEnabled,
			AiModels:          models,
		},
		History: history,
	}
}
