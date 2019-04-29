package ui

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/gdamore/tcell"
	"github.com/lonng/0x81/directive"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func (ui *UI) handleEvents() {
	ui.app.SetInputCapture(ui.handleApp)
	ui.history.SetInputCapture(ui.handleHistory)
	ui.sqlStmt.SetDoneFunc(ui.sqlStmtDone)
	ui.sqlStmt.SetInputCapture(ui.sqlStmtKey)
	ui.mysqlPanel.SetInputCapture(ui.esc)
	ui.tidbPanel.SetInputCapture(ui.esc)
}

func (ui *UI) handleApp(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() != tcell.KeyTAB {
		return event
	}

	focusables := ui.focusables
	app := ui.app
	current := app.GetFocus()
	var index = 0
	for i, f := range focusables {
		if f == current {
			index = i
			break
		}
	}
	index += 1
	index %= len(focusables)
	app.SetFocus(focusables[index])
	return event
}

func (ui *UI) esc(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() != tcell.KeyESC {
		return event
	}
	ui.app.SetFocus(ui.sqlStmt)
	return event
}

func (ui *UI) handleHistory(event *tcell.EventKey) *tcell.EventKey {
	app := ui.app
	sqlStmt := ui.sqlStmt
	history := ui.history
	switch event.Key() {
	case tcell.KeyESC:
		app.SetFocus(sqlStmt)
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		index := ui.history.GetCurrentItem()
		if !ui.recorder.Delete(index) {
			break
		}
		ui.renderHistory()
		if index < ui.history.GetItemCount() {
			ui.history.SetCurrentItem(index)
		} else if index > 0 {
			ui.history.SetCurrentItem(index - 1)
		}
	case tcell.KeyEnter:
		text, _ := history.GetItemText(history.GetCurrentItem())
		if strings.HasSuffix(text, "*/") {
			lastIndex := strings.LastIndex(text, "/*->")
			text = text[:lastIndex]
		}
		sqlStmt.SetText(text[35:])
		app.SetFocus(sqlStmt)
	}
	return event
}

type queryResult struct {
	result   *sql.Rows
	error    error
	duration time.Duration
	rowcount int
}

func (result *queryResult) stat() string {
	if result.error != nil {
		return fmt.Sprintf("failure (%.3f sec)", result.duration.Seconds())
	}
	return fmt.Sprintf("%d row in set (%.3f sec)", result.rowcount, result.duration.Seconds())
}

func (result *queryResult) content() string {
	if result.error != nil {
		return result.error.Error()
	}
	cols, err := result.result.Columns()
	if err != nil {
		return err.Error()
	}
	var allRows [][][]byte
	for result.result.Next() {
		var columns = make([][]byte, len(cols))
		var pointer = make([]interface{}, len(cols))
		for i := range columns {
			pointer[i] = &columns[i]
		}
		err := result.result.Scan(pointer...)
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

func (result *queryResult) close() {
	if result.result == nil {
		return
	}
	result.result.Close()
}

func (ui *UI) query(text string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mysqlResultCh := make(chan queryResult)
	tidbResultCh := make(chan queryResult)
	go func() {
		start := time.Now()
		rows, err := ui.mysql.QueryContext(ctx, text)
		duration := time.Since(start)
		mysqlResultCh <- queryResult{result: rows, error: err, duration: duration}
	}()

	go func() {
		start := time.Now()
		rows, err := ui.tidb.QueryContext(ctx, text)
		duration := time.Since(start)
		tidbResultCh <- queryResult{result: rows, error: err, duration: duration}
	}()

	mysqlResult := <-mysqlResultCh
	tidbResult := <-tidbResultCh
	defer mysqlResult.close()
	defer tidbResult.close()

	// Highlight diff
	mysqlContent, tidbContent := mysqlResult.content(), tidbResult.content()
	if mysqlResult.error == nil && tidbResult.error == nil {
		patch := diffmatchpatch.New()
		diff := patch.DiffMain(mysqlContent, tidbContent, false)
		var newMySQLContent, newTiDBContent bytes.Buffer
		for _, d := range diff {
			switch d.Type {
			case diffmatchpatch.DiffEqual:
				newMySQLContent.WriteString(d.Text)
				newTiDBContent.WriteString(d.Text)
			case diffmatchpatch.DiffDelete:
				newMySQLContent.WriteString("[red]" + d.Text + "[white]")
			case diffmatchpatch.DiffInsert:
				newTiDBContent.WriteString("[green]" + d.Text + "[white]")
			}
		}
		mysqlContent = newMySQLContent.String()
		tidbContent = newTiDBContent.String()
	}
	mysqlStat, tidbStat := mysqlResult.stat(), tidbResult.stat()
	fmt.Fprintln(ui.mysqlPanel, mysqlContent)
	fmt.Fprintln(ui.mysqlPanel, mysqlStat+"\n")
	fmt.Fprintln(ui.tidbPanel, tidbContent)
	fmt.Fprintln(ui.tidbPanel, tidbStat+"\n")
	ui.tidbPanel.Highlight()
}

func (ui *UI) sqlStmtDone(key tcell.Key) {
	if key != tcell.KeyEnter {
		return
	}

	query := strings.TrimSpace(ui.sqlStmt.GetText())
	if query == "" {
		return
	}

	// Parse directive if query start with `!`
	if query[0] == '!' {
		text := query[1:]
		temp, err := template.New("query-template").Funcs(directive.Functions).Parse(text)
		if err != nil {
			ui.recordHistory(fmt.Sprintf("%s /*->[red] %s[white]*/", query, err.Error()))
			return
		}
		out := bytes.Buffer{}
		if err := temp.Execute(&out, nil); err != nil {
			ui.recordHistory(fmt.Sprintf("%s /*->[red] %s[white]*/", query, err.Error()))
		}
		ui.query(out.String())
	} else {
		ui.query(query)
	}
	ui.recordHistory(query)
}

func (ui *UI) recordHistory(query string) {
	ui.recorder.Record(time.Now(), query)
	ui.renderHistory()
	ui.history.SetCurrentItem(0)
	ui.sqlStmt.SetText("")
}

func (ui *UI) renderHistory() {
	history := ui.history
	history.Clear()
	items := ui.recorder.Items()
	for _, item := range items {
		history.AddItem(item.String(), "", 0, nil)
	}
}

func (ui *UI) sqlStmtKey(event *tcell.EventKey) *tcell.EventKey {
	app := ui.app
	history := ui.history
	if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown {
		app.SetFocus(history)
	}
	return event
}
