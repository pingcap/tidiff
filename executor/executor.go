package executor

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/pingcap/tidiff/directive"
)

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

func (e *Executor) Open() error {
	if atomic.AddInt32(&e.started, 1) != 1 {
		return errors.New("executor started")
	}
	mysql, err := sql.Open("mysql", e.MySQLConfig.DSN())
	if err != nil {
		return err
	}
	tidb, err := sql.Open("mysql", e.TiDBConfig.DSN())
	if err != nil {
		return err
	}
	e.mysql = mysql
	e.tidb = tidb
	return nil
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
