package uimode

import (
	"database/sql"

	"github.com/lonng/tidiff/executor"
	"github.com/lonng/tidiff/history"
	"github.com/rivo/tview"
)

type UI struct {
	app      *tview.Application
	recorder *history.Recorder
	executor *executor.Executor
	mysql    *sql.DB
	tidb     *sql.DB

	// panels
	sqlStmt    *tview.InputField
	history    *tview.List
	mysqlPanel *tview.TextView
	tidbPanel  *tview.TextView

	focusables []tview.Primitive
}

func New(recorder *history.Recorder, exec *executor.Executor, mysql, tidb *sql.DB) *UI {
	return &UI{
		app:      tview.NewApplication(),
		recorder: recorder,
		executor: exec,
		mysql:    mysql,
		tidb:     tidb,
	}
}

func (ui UI) Serve() (err error) {
	err = ui.recorder.Load()
	if err != nil {
		return err
	}
	ui.layout()
	ui.handleEvents()
	return ui.app.Run()
}
