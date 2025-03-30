package stream

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bluenviron/gortsplib/v4/pkg/ringbuffer"
	"github.com/xaionaro-go/mediamtx/pkg/counterdumper"
	"github.com/xaionaro-go/mediamtx/pkg/logger"
)

var Logger slog.Logger

type streamReader struct {
	queueSize int
	parent    logger.Writer

	buffer          *ringbuffer.RingBuffer
	started         bool
	discardedFrames *counterdumper.CounterDumper

	// out
	err chan error
}

func (w *streamReader) initialize() {
	buffer, err := ringbuffer.New(uint64(w.queueSize))
	if err != nil {
		Logger.Log(context.TODO(), slog.LevelError, "unable to initialize the ring buffer", slog.String("error", err.Error()))
	}
	w.buffer = buffer
	w.err = make(chan error)
}

func (w *streamReader) start() {
	w.started = true

	w.discardedFrames = &counterdumper.CounterDumper{
		OnReport: func(val uint64) {
			w.parent.Log(logger.Warn, "connection is too slow, discarding %d %s",
				val,
				func() string {
					if val == 1 {
						return "frame"
					}
					return "frames"
				}())
		},
	}
	w.discardedFrames.Start()

	go w.run()
}

func (w *streamReader) stop() {
	w.buffer.Close()

	if w.started {
		w.discardedFrames.Stop()
		<-w.err
	}
}

func (w *streamReader) error() chan error {
	return w.err
}

func (w *streamReader) run() {
	w.err <- w.runInner()
	close(w.err)
}

func (w *streamReader) runInner() error {
	for {
		cb, ok := w.buffer.Pull()
		if !ok {
			return fmt.Errorf("terminated")
		}

		err := cb.(func() error)()
		if err != nil {
			return err
		}
	}
}

func (w *streamReader) push(cb func() error) {
	ok := w.buffer.Push(cb)
	if !ok {
		w.discardedFrames.Increase()
	}
}
