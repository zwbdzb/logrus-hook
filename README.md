# BufferedWriterHook
logrus' logs are known to be synchronous, logrus may occur Performance issues in concurrent scenarios.

This package provides asynchronous buffered log-writing capability for logrus.

#### Usage
the code componment is  `BufferedWriterHook`,
the `defaultBufferSize` is 256kb, `defaultFlushInterval` is 10 s
```
    l := logrus.New()
	l.SetLevel(logrus.InfoLevel)
	l.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logf, err := os.OpenFile("./log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	defer logf.Close()

	l.SetOutput(io.Discard) // Send all logs to nowhere by default

	fileAndStdoutWriter := io.MultiWriter(logf, os.Stdout)
	ws := &BufferedWriterHook{Writer: fileAndStdoutWriter}
	defer ws.Stop()  // please call [stop] function when program  coming to an end.
	l.AddHook(ws)
```

### BenchMark

the benchmark code is in the test source code.

 ```
go test  -bench=Benchmark* . -benchmem
level=info msg="test2023-07-21T20:38:30+08:00"
goos: windows
goarch: amd64
pkg: github.com/zwbdzb/logrus-bufferedWriterHook
cpu: 11th Gen Intel(R) Core(TM) i7-1165G7 @ 2.80GHz
BenchmarkAsyncWriter-8           1031562              1127 ns/op             464 B/op         13 allocs/op
BenchmarkSyncWriter-8              64758             16479 ns/op             496 B/op         15 allocs/op
PASS
ok      github.com/zwbdzb/logrus-bufferedWriterHook     3.638s
 ```


##  logrus-mysql-hook

log to mysql for aduit，


