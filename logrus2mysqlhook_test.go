package logrushook

import (
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"xorm.io/xorm"
)

var engine *xorm.Engine = nil

func InitDB(user, pwd, hp, db string) error {
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", user, pwd, hp, db)
	var err error
	engine, err = xorm.NewEngine("mysql", connStr)
	if err != nil {
		return err
	}
	engine.ShowSQL(true)
	//连接池最大空闲
	engine.SetMaxIdleConns(10)
	//最大连接数
	engine.SetMaxOpenConns(50)
	return err
}

func TestMysql(t *testing.T) {
	InitDB("tvsn", "vvq1+KLbooj", "10.100.41.110:3980", "teinfra_tvs_sync_naming")
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
	ws := &Logrus2MysqlHook{Engine: engine}
	defer ws.Stop() // 等待子协程结束
	l.AddHook(ws)
	l.WithField("user", "test name").WithField("uid", "12121").WithField("method", "test post").WithField("path", "test path").WithField("body", "test body").WithField("status", 200).Info()
	time.Sleep(time.Second * 10)
}
