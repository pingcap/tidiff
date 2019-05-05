package uimode

import (
	"github.com/pingcap/tidiff/executor"
	"github.com/pingcap/tidiff/history"
	"github.com/rivo/tview"
)

type UI struct {
	app      *tview.Application
	recorder *history.Recorder
	executor *executor.Executor

	// panels
	sqlStmt    *tview.InputField
	history    *tview.List
	mysqlPanel *tview.TextView
	tidbPanel  *tview.TextView

	focusables []tview.Primitive
}

func New(recorder *history.Recorder, exec *executor.Executor) *UI {
	return &UI{
		app:      tview.NewApplication(),
		recorder: recorder,
		executor: exec,
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
