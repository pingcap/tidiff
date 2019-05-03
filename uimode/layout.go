package uimode

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func (ui *UI) layout() {
	// Header panels (sql statement input field and history panel)
	sqlStmt := tview.NewInputField()
	sqlStmt.SetLabel("SQL> ").SetFieldBackgroundColor(tcell.ColorBlack)
	history := tview.NewList()
	history.SetBorder(true).SetTitle("History")
	history.ShowSecondaryText(false)
	history.SetBorderPadding(0, 0, 1, 1)
	history.SetSelectedFocusOnly(true)
	header := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(history, 0, 1, false).
		AddItem(sqlStmt, 1, 1, false)

	// Result sets panels (MySQL result and TiDB results)
	mysqlPanel := tview.NewTextView()
	mysqlPanel.SetBorder(true).SetTitle("MySQL").SetBorderPadding(0, 0, 1, 1)
	mysqlPanel.SetDynamicColors(true).SetRegions(true)
	tidbPanel := tview.NewTextView()
	tidbPanel.SetBorder(true).SetTitle("TiDB").SetBorderPadding(0, 0, 1, 1)
	tidbPanel.SetDynamicColors(true).SetRegions(true)
	resultSets := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(mysqlPanel, 0, 1, false).
		AddItem(tidbPanel, 0, 1, false)

	// Key `TAB` will switch focus around focusable widgets, all panels which want get focus
	// on `TAB` hit should be placed in `ui.focusables` slice
	ui.sqlStmt = sqlStmt
	ui.history = history
	ui.mysqlPanel = mysqlPanel
	ui.tidbPanel = tidbPanel
	ui.focusables = []tview.Primitive{sqlStmt, history, mysqlPanel, tidbPanel}

	// Restore history query
	if histories := ui.recorder.Items(); len(histories) > 0 {
		for _, h := range histories {
			if h.Text == "" {
				continue
			}
			history.AddItem(h.String(), "", 0, nil)
		}
		history.SetCurrentItem(0)
	}

	// Display MySQL/TiDB version information
	ui.query("select version()")

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(resultSets, 0, 5, false).
		AddItem(header, 0, 2, false)

	ui.app.SetRoot(container, true).SetFocus(sqlStmt)
}
