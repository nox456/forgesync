package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

const RefreshTimeout = 2 * time.Second

var ErrRefreshTimeout = errors.New("update check timed out")

type LatestFinder interface {
	LatestVersion(ctx context.Context) (string, error)
}

type Notifier struct {
	Current string
	Finder  LatestFinder
	Store   *Store
	Out     io.Writer
	Now     func() time.Time
	Timeout time.Duration
}

type result struct {
	latest string
	err    error
}

type Refresh struct {
	n      *Notifier
	state  State
	ctx    context.Context
	cancel context.CancelFunc
	done   chan result
}

func (n *Notifier) Start() (*Refresh, error) {
	state, err := n.Store.Load()
	if err != nil {
		return nil, err
	}

	if IsNewer(n.Current, state.LatestVersion) {
		fmt.Fprintln(n.Out, NewNotice(state.LatestVersion))
	}

	if !ShouldCheck(state, n.Now(), state.Interval()) {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), n.Timeout)
	done := make(chan result, 1)
	go func() {
		latest, err := n.Finder.LatestVersion(ctx)
		done <- result{latest, err}
	}()
	return &Refresh{n: n, state: state, ctx: ctx, cancel: cancel, done: done}, nil
}

func (r *Refresh) Wait() error {
	defer r.cancel()

	select {
	case res := <-r.done:
		return r.save(res)
	case <-r.ctx.Done():
		return ErrRefreshTimeout
	}
}

func (r *Refresh) save(res result) error {
	if res.err == nil {
		r.state.LatestVersion = res.latest
		r.state.LastCheckedAt = r.n.Now()
		return r.n.Store.Save(r.state)
	}

	if errors.Is(res.err, ErrNoRelease) {
		r.state.LastCheckedAt = r.n.Now()
		return r.n.Store.Save(r.state)
	}

	return res.err
}
