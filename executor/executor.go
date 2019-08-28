package executor

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/pingcap/tidiff/directive"
)

const DefaultRetryCnt = 1

type Executor struct {
	MySQLConfig *Config
	TiDBConfig  *Config
	mysql       *sql.DB
	tidb        *sql.DB
	started     int32
}

func NewExecutor(mysql, tidb *Config) *Executor {
	return &Executor{MySQLConfig: mysql, TiDBConfig: tidb}
}

func (e *Executor) Open(retryCnt int) error {
	if atomic.AddInt32(&e.started, 1) != 1 {
		return errors.New("executor started")
	}
	mysql, err := openDBWithRetry("mysql", e.MySQLConfig.DSN(), retryCnt)
	if err != nil {
		return err
	}
	tidb, err := openDBWithRetry("mysql", e.TiDBConfig.DSN(), retryCnt)
	if err != nil {
		return err
	}
	e.mysql = mysql
	e.tidb = tidb
	return nil
}

// openDBWithRetry opens a database specified by its database driver name and a
// driver-specific data source name. And it will do some retries if the connection fails.
func openDBWithRetry(driverName, dataSourceName string, retryCnt int) (mdb *sql.DB, err error) {
	startTime := time.Now()
	sleepTime := time.Millisecond * 500
	for i := 0; i < retryCnt; i++ {
		mdb, err = sql.Open(driverName, dataSourceName)
		if err != nil {
			fmt.Printf("open db %s failed, retry count %d err %v\n", dataSourceName, i, err)
			time.Sleep(sleepTime)
			continue
		}
		err = mdb.Ping()
		if err == nil {
			break
		}
		fmt.Printf("ping db %s failed, retry count %d err %v\n", dataSourceName, i, err)
		mdb.Close()
		time.Sleep(sleepTime)
	}
	if err != nil {
		fmt.Printf("open db %s failed %v, take time %v\n", dataSourceName, err, time.Since(startTime))
		return nil, err
	}

	return
}

func q(ctx context.Context, db *sql.DB, query string, ch chan *QueryResult) {
	go func() {
		start := time.Now()
		rows, err := db.QueryContext(ctx, query)
		duration := time.Since(start)
		ch <- &QueryResult{Result: rows, Error: err, Rendered: query, duration: duration}
	}()
}

func (e *Executor) query(query string) (*QueryResult, *QueryResult) {
	ctx := context.Background()
	mysqlResultCh := make(chan *QueryResult)
	tidbResultCh := make(chan *QueryResult)

	q(ctx, e.mysql, query, mysqlResultCh)
	q(ctx, e.tidb, query, tidbResultCh)

	mysqlResult := <-mysqlResultCh
	tidbResult := <-tidbResultCh
	return mysqlResult, tidbResult
}

func (e *Executor) Query(query string) (*QueryResult, *QueryResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil, errors.New("empty query")
	}

	var mysqlResult, tidbResult *QueryResult
	// Parse directive if query start with `!`
	if len(query) > 1 && query[0] == '!' {
		text := strings.TrimLeft(query, "!")
		temp, err := template.New("template").Funcs(directive.Functions).Parse(text)
		if err != nil {
			return nil, nil, err
		}
		out := bytes.Buffer{}
		if err := temp.Execute(&out, nil); err != nil {
			return nil, nil, err
		}
		text = strings.TrimSpace(out.String())
		mysqlResult, tidbResult = e.query(text)
	} else {
		mysqlResult, tidbResult = e.query(query)
	}
	return mysqlResult, tidbResult, nil
}
