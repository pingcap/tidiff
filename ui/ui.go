package ui

import (
	"database/sql"

	"github.com/lonng/0x81/history"
	"github.com/rivo/tview"
)

type UI struct {
	app      *tview.Application
	recorder *history.Recorder
	mysql    *sql.DB
	tidb     *sql.DB

	// panels
	sqlStmt    *tview.InputField
	history    *tview.List
	mysqlPanel *tview.TextView
	tidbPanel  *tview.TextView

	focusables []tview.Primitive
}

func New(recorder *history.Recorder, mysql, tidb *sql.DB) *UI {
	return &UI{
		app:      tview.NewApplication(),
		recorder: recorder,
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
