package executor

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type QueryResult struct {
	Result   *sql.Rows
	Error    error
	Rendered string
	duration time.Duration
	rowcount int
}

func (result *QueryResult) Stat() string {
	if result.Error != nil {
		return fmt.Sprintf("failure (%.3f sec)", result.duration.Seconds())
	}
	return fmt.Sprintf("%d row in set (%.3f sec)", result.rowcount, result.duration.Seconds())
}

func (result *QueryResult) Content() string {
	if result.Error != nil {
		return result.Error.Error()
	}
	cols, err := result.Result.Columns()
	if err != nil {
		return err.Error()
	}
	var allRows [][][]byte
	for result.Result.Next() {
		var columns = make([][]byte, len(cols))
		var pointer = make([]interface{}, len(cols))
		for i := range columns {
			pointer[i] = &columns[i]
		}
		err := result.Result.Scan(pointer...)
		if err != nil {
			return err.Error()
		}
		allRows = append(allRows, columns)
		result.rowcount++
	}
	if result.rowcount < 1 {
		return "Empty set"
	}

	// Calculate the max column length
	var colLength []int
	for _, c := range cols {
		colLength = append(colLength, len(c))
	}
	for _, row := range allRows {
		for n, col := range row {
			if l := len(col); colLength[n] < l {
				colLength[n] = l
			}
		}
	}
	// The total length
	var total = len(cols) - 1
	for index := range colLength {
		colLength[index] += 2 // Value will wrap with space
		total += colLength[index]
	}

	var lines []string
	var push = func(line string) {
		lines = append(lines, line)
	}

	// Write table header
	var header string
	for index, col := range cols {
		length := colLength[index]
		padding := length - 1 - len(col)
		if index == 0 {
			header += "|"
		}
		header += " " + col + strings.Repeat(" ", padding) + "|"
	}
	splitLine := "+" + strings.Repeat("-", total) + "+"
	push(splitLine)
	push(header)
	push(splitLine)

	// Write rows data
	for _, row := range allRows {
		var line string
		for index, col := range row {
			length := colLength[index]
			padding := length - 1 - len(col)
			if index == 0 {
				line += "|"
			}
			line += " " + string(col) + strings.Repeat(" ", padding) + "|"
		}
		push(line)
	}
	push(splitLine)
	return strings.Join(lines, "\n")
}

func (result *QueryResult) Close() {
	if result.Result == nil {
		return
	}
	result.Result.Close()
}
