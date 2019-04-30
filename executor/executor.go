package executor

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"text/template"
	"time"

	"github.com/lonng/tidiff/directive"
)

type Executor struct {
	mysql *sql.DB
	tidb  *sql.DB
}

func NewExecutor(mysql, tidb *sql.DB) *Executor {
	return &Executor{mysql: mysql, tidb: tidb}
}

func (e *Executor) query(query string) (*QueryResult, *QueryResult) {
	ctx := context.Background()
	mysqlResultCh := make(chan *QueryResult)
	tidbResultCh := make(chan *QueryResult)
	go func() {
		start := time.Now()
		rows, err := e.mysql.QueryContext(ctx, query)
		duration := time.Since(start)
		mysqlResultCh <- &QueryResult{Result: rows, Error: err, Rendered: query, duration: duration}
	}()

	go func() {
		start := time.Now()
		rows, err := e.tidb.QueryContext(ctx, query)
		duration := time.Since(start)
		tidbResultCh <- &QueryResult{Result: rows, Error: err, Rendered: query, duration: duration}
	}()

	mysqlResult := <-mysqlResultCh
	tidbResult := <-tidbResultCh
	return mysqlResult, tidbResult
}

func (e *Executor) Query(query string) (*QueryResult, *QueryResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil, errors.New("emtpy query")
	}

	var mysqlResult, tidbResult *QueryResult
	// Parse directive if query start with `!`
	if len(query) > 1 && query[0] == '!' {
		text := strings.TrimLeft(query, "!")
		temp, err := template.New("query-template").Funcs(directive.Functions).Parse(text)
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
