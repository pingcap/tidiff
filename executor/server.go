package executor

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DoDiff executes sqls in the driver configured in exec. If there is an inconsistent sql execution result,
// it will directly return an error, and the error message contains inconsistencies.
// If the execution results are the same, no error is returned.
// handleResultsFn is used to do special processing on the results after executing each statement.
func (exec *Executor) DoDiff(sqls []string, handleResultsFn func(strs []string)) error {
	if len(sqls) == 0 {
		return nil
	}

	for _, sql := range sqls {
		if err := exec.diffExecResult(sql, handleResultsFn); err != nil {
			return err
		}
	}

	return nil
}

func (exec *Executor) diffExecResult(query string, handleResultsFn func(str []string)) error {
	mysqlResult, tidbResult, err := exec.Query(query)
	if err != nil {
		return err
	}
	defer mysqlResult.Close()
	defer tidbResult.Close()
	mysqlContent, tidbContent := mysqlResult.Content(), tidbResult.Content()
	if mysqlResult.Error == nil && tidbResult.Error == nil {
		green := color.New(color.FgGreen).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		patch := diffmatchpatch.New()
		diff := patch.DiffMain(mysqlContent, tidbContent, false)
		var newMySQLContent, newTiDBContent bytes.Buffer
		for _, d := range diff {
			switch d.Type {
			case diffmatchpatch.DiffEqual:
				newMySQLContent.WriteString(d.Text)
				newTiDBContent.WriteString(d.Text)
			case diffmatchpatch.DiffDelete:
				newMySQLContent.WriteString(red(d.Text))
			case diffmatchpatch.DiffInsert:
				newTiDBContent.WriteString(green(d.Text))
			}
		}
		mysqlContent = newMySQLContent.String()
		tidbContent = newTiDBContent.String()
	}

	logQuery := query
	if strings.HasPrefix(query, "!!") {
		logQuery = mysqlResult.Rendered
	}

	if handleResultsFn != nil {
		handleResultsFn([]string{mysqlContent, tidbContent})
	}
	if mysqlContent == tidbContent {
		return nil
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("MySQL(%s)> %s\n", exec.MySQLConfig.Address(), logQuery))
	if mysqlContent != "" {
		buf.WriteString(mysqlContent)
	}
	buf.WriteString(mysqlResult.Stat() + "\n")
	buf.WriteString(fmt.Sprintf("TiDB(%s)> %s\n", exec.TiDBConfig.Address(), logQuery))
	if tidbContent != "" {
		buf.WriteString(tidbContent)
	}
	buf.WriteString(tidbResult.Stat() + "\n")
	return errors.New(buf.String())
}
