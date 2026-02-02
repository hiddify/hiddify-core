package hcore

import (
	"time"
)

type Debouncer struct {
	delay time.Duration
	timer *time.Timer
}

func NewDebouncer(d time.Duration) *Debouncer {
	t := time.NewTimer(d)
	t.Stop()
	return &Debouncer{
		delay: d,
		timer: t,
	}
}

func (d *Debouncer) Hit() {
	if !d.timer.Stop() {
		select {
		case <-d.timer.C:
		default:
		}
	}
	d.timer.Reset(d.delay)
}

func (d *Debouncer) C() <-chan time.Time {
	return d.timer.C
}

func (d *Debouncer) Stop() {
	d.timer.Stop()
}
