package logrushook

import (
	"bufio"
	"io"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	_defaultBufferSize = 256 * 1024 // 256 kB

	_defaultFlushInterval = 10 * time.Second
)

type BufferedWriterHook struct {
	Writer    io.Writer
	LogLevels []logrus.Level

	// Size specifies the maximum amount of data the writer will buffered
	// before flushing.
	//
	// Defaults to 256 kB if unspecified.
	Size int

	// FlushInterval specifies how often the writer should flush data if
	// there have been no writes.
	//
	// Defaults to 30 seconds if unspecified.
	FlushInterval time.Duration

	// Clock, if specified, provides control of the source of time for the
	// writer.
	//
	// Defaults to the system clock.
	Clock Clock

	// unexported fields for state
	mu          sync.Mutex
	initialized bool // whether initialize() has run
	stopped     bool // whether Stop() has run
	writer      *bufio.Writer
	ticker      *time.Ticker
	stop        chan struct{} // closed when flushLoop should stop
	done        chan struct{} // closed when flushLoop has stopped
}

func (s *BufferedWriterHook) initialize() {
	size := s.Size
	if size == 0 {
		size = _defaultBufferSize
	}

	flushInterval := s.FlushInterval
	if flushInterval == 0 {
		flushInterval = _defaultFlushInterval
	}

	if s.Clock == nil {
		s.Clock = DefaultClock
	}

	s.ticker = s.Clock.NewTicker(flushInterval)
	s.writer = bufio.NewWriterSize(s.Writer, size)
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.initialized = true
	go s.flushLoop()
}

// Hook function when logrus is triggered to write logs
func (s *BufferedWriterHook) Fire(entry *logrus.Entry) error {
	bs, err := entry.Bytes()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		s.initialize()
	}

	// To avoid partial writes from being flushed, we manually flush the existing buffer if:
	// * The current write doesn't fit into the buffer fully, and
	// * The buffer is not empty (since bufio will not split large writes when the buffer is empty)
	if len(bs) > s.writer.Available() && s.writer.Buffered() > 0 {
		if err := s.writer.Flush(); err != nil {
			return err
		}
	}
	_, err = s.writer.Write(bs)
	return err
}

// Decide which log levels will use the hook
func (s *BufferedWriterHook) Levels() []logrus.Level {
	logLevels := s.LogLevels
	if len(logLevels) == 0 {
		logLevels = []logrus.Level{
			logrus.InfoLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
			logrus.FatalLevel,
		}
	}
	s.LogLevels = logLevels
	return s.LogLevels
}

// Sync flushes buffered log data into disk directly.
func (s *BufferedWriterHook) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	if s.initialized {
		err = s.writer.Flush()
	}

	return err
}

// flushLoop flushes the buffer at the configured interval until Stop is
// called.
func (s *BufferedWriterHook) flushLoop() {
	defer close(s.done)

	for {
		select {
		case <-s.ticker.C:
			// we just simply ignore error here
			// because the underlying bufio writer stores any errors
			// and we return any error from Sync() as part of the close
			_ = s.Sync()
		case <-s.stop:
			return
		}
	}
}

// Stop closes the buffer, cleans up background goroutines, and flushes
// remaining unwritten data.
func (s *BufferedWriterHook) Stop() (err error) {
	var stopped bool

	// Critical section.
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		if !s.initialized {
			return
		}

		stopped = s.stopped
		if stopped {
			return
		}
		s.stopped = true

		s.ticker.Stop()
		close(s.stop) // tell flushLoop to stop
		<-s.done      // and wait until it has
	}()

	// Don't call Sync on consecutive Stops.
	if !stopped {
		err = s.Sync()
	}

	return err
}

// ---

// time. This clock uses the system clock for all operations.
var DefaultClock = systemClock{}

// Clock is a source of time for logged entries.
type Clock interface {
	// Now returns the current local time.
	Now() time.Time

	// NewTicker returns *time.Ticker that holds a channel
	// that delivers "ticks" of a clock.
	NewTicker(time.Duration) *time.Ticker
}

// systemClock implements default Clock that uses system time.
type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

func (systemClock) NewTicker(duration time.Duration) *time.Ticker {
	return time.NewTicker(duration)
}
