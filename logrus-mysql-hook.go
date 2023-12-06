package logrushook

import (
	"sync"

	"github.com/sirupsen/logrus"
	"xorm.io/xorm"
)

const aduitTable = "operations_aduit"

type Logrus2MysqlHook struct {
	Engine      *xorm.Engine
	LogLevels   []logrus.Level
	mu          sync.Mutex
	initialized bool
	stopped     bool          // whether Stop() has run
	stop        chan struct{} // closed when flushLoop should stop , 通知协程结束， 发出信号
	done        chan struct{} // closed when flushLoop has stopped ，等待协程结束， 得到信号
	queue       chan *logrus.Entry
}

func (s *Logrus2MysqlHook) initialize() {
	s.queue = make(chan *logrus.Entry, 30)
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.initialized = true
	go s.flush()
}

func (s *Logrus2MysqlHook) Fire(entry *logrus.Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		s.initialize()
	}
	s.queue <- entry
	return nil
}

// 决定哪些Loglevel能触发该hook
func (s *Logrus2MysqlHook) Levels() []logrus.Level {
	logLevels := s.LogLevels
	if len(logLevels) == 0 {
		logLevels = []logrus.Level{
			logrus.InfoLevel,
			logrus.ErrorLevel,
			logrus.FatalLevel,
		}
	}
	s.LogLevels = logLevels
	return s.LogLevels
}

// Sync flushes buffered log data into disk directly.
func (s *Logrus2MysqlHook) Sync(entry *logrus.Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	if s.initialized {
		val := make(map[string]interface{}, 10)
		for k, v := range entry.Data {
			val[k] = v
		}
		val["time"] = entry.Time // .Format(time.RFC3339)
		_, err := s.Engine.Table(aduitTable).Insert(val)
		if err != nil {
			return err
		}
		return nil
	}
	return err
}

func (s *Logrus2MysqlHook) flush() {
	defer close(s.done)

	for {
		select {
		case e := <-s.queue:
			_ = s.Sync(e)
		case <-s.stop:
			return
		}
	}
}

func (s *Logrus2MysqlHook) Stop() (err error) {
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

		close(s.stop) // tell flush to stop
		<-s.done      // and wait until it has
	}()
	return err
}
