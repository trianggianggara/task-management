package job

import (
	"context"
	"log/slog"
	"time"
)

type Job interface {
	Name() string
	Interval() time.Duration
	Run(ctx context.Context) error
}

type Runner struct {
	jobs []Job
}

func NewRunner(jobs ...Job) *Runner {
	return &Runner{jobs: jobs}
}

func (r *Runner) Add(j Job) {
	r.jobs = append(r.jobs, j)
}

func (r *Runner) Start(ctx context.Context) {
	for _, j := range r.jobs {
		go r.run(ctx, j)
	}
	<-ctx.Done()
}

func (r *Runner) run(ctx context.Context, j Job) {
	slog.Info("job started", "name", j.Name(), "interval", j.Interval().String())

	if err := j.Run(ctx); err != nil {
		slog.Error("job initial run failed", "name", j.Name(), "error", err)
	}

	ticker := time.NewTicker(j.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("job stopped", "name", j.Name())
			return
		case <-ticker.C:
			if err := j.Run(ctx); err != nil {
				slog.Error("job failed", "name", j.Name(), "error", err)
			}
		}
	}
}
