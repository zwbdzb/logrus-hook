package logrushook

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func ExampleHook_default() {
	l := logrus.New()
	l.SetLevel(logrus.InfoLevel)
	l.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	l.SetOutput(io.Discard) // Send all logs to nowhere by default

	ws := &BufferedWriterHook{Writer: os.Stdout}
	l.ExitFunc = func(code int) {
		ws.Stop()
	}
	l.AddHook(ws)

	l.Info("test2")
	l.Warn("test3")
	l.Error("test4")

	// Output:
	// level=info msg=test2
	// level=warning msg=test3
	// level=error msg=test4
}

func TestWriteLog(t *testing.T) {

	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	l.SetOutput(io.Discard) // Send all logs to nowhere by default
	bh := &BufferedWriterHook{Writer: os.Stdout}
	defer bh.Stop()

	err := bh.Fire(&logrus.Entry{Logger: l, Level: logrus.InfoLevel, Message: "test" + time.Now().Format(time.RFC3339)})
	if err != nil {
		t.Error(t.Name() + " FAIL")
	}
}

//  go test -run  TestWriteLog  -count=1

func BenchmarkAsyncWriter(b *testing.B) {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logf, err := os.OpenFile("./log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	defer logf.Close()

	bh := &BufferedWriterHook{Writer: logf}
	defer bh.Stop()

	b.ResetTimer() // 重置计时器，忽略前面的准备时间
	for n := 0; n < b.N; n++ {
		err := bh.Fire(&logrus.Entry{Logger: l, Level: logrus.InfoLevel, Message: "test" + time.Now().Format(time.RFC3339)})
		if err != nil {
			b.Error(b.Name() + " FAIL")
		}
	}
}

func BenchmarkSyncWriter(b *testing.B) {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logf, err := os.OpenFile("./log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	defer logf.Close()

	l.SetOutput(logf)

	b.ResetTimer()             // 重置计时器，忽略前面的准备时间
	for n := 0; n < b.N; n++ { // B.N指定了迭代次数， 不固定，动态分配，至少1s
		l.Info("1test" + time.Now().Format(time.RFC3339))
	}
}

// go  test  -bench=. -count=3
