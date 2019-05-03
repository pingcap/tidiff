package uimode

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell"
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

func (ui *UI) query(query string) {
	if query == "" {
		return
	}

	mysqlResult, tidbResult, err := ui.executor.Query(query)
	if err != nil {
		ui.recordHistory(fmt.Sprintf("%s /*->[red] %s[white]*/", query, err.Error()))
		return
	}
	defer mysqlResult.Close()
	defer tidbResult.Close()

	// Highlight diff
	mysqlContent, tidbContent := mysqlResult.Content(), tidbResult.Content()
	if mysqlResult.Error == nil && tidbResult.Error == nil {
		patch := diffmatchpatch.New()
		diff := patch.DiffMain(mysqlContent, tidbContent, false)
		if ui.recorder.IsDiffEnable() {
			ui.recorder.LogDiff(patch.DiffPrettyText(diff))
		}
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

	logQuery := query
	if strings.HasPrefix(query, "!!") {
		logQuery = mysqlResult.Rendered
	}
	fmt.Fprintln(ui.mysqlPanel, fmt.Sprintf("MySQL(%s)> %s", ui.executor.MySQLConfig.Address(), logQuery))
	fmt.Fprintln(ui.tidbPanel, fmt.Sprintf("TiDB(%s)> %s", ui.executor.TiDBConfig.Address(), logQuery))
	if mysqlContent != "" {
		fmt.Fprintln(ui.mysqlPanel, mysqlContent)
	}
	fmt.Fprintln(ui.mysqlPanel, mysqlResult.Stat()+"\n")
	if tidbContent != "" {
		fmt.Fprintln(ui.tidbPanel, tidbContent)
	}
	fmt.Fprintln(ui.tidbPanel, tidbResult.Stat()+"\n")
}

func (ui *UI) sqlStmtDone(key tcell.Key) {
	if key != tcell.KeyEnter {
		return
	}
	query := strings.TrimSpace(ui.sqlStmt.GetText())
	ui.query(query)
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
