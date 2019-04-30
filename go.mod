module github.com/lonng/tidiff

require (
	github.com/fatih/color v1.7.1-0.20181010231311-3f9d52f7176a
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/gdamore/tcell v1.1.1
	github.com/go-sql-driver/mysql v1.4.1-0.20190423112050-d0a548181995
	github.com/lucasb-eyer/go-colorful v1.0.2 // indirect
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-isatty v0.0.7 // indirect
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/rivo/tview v0.0.0-20190406182340-90b4da1bd64c
	github.com/rivo/uniseg v0.0.0-20190313204849-f699dde9c340 // indirect
	github.com/sergi/go-diff v1.0.1-0.20180205163309-da645544ed44
	github.com/urfave/cli v1.20.1-0.20190203184040-693af58b4d51
	golang.org/x/text v0.3.2 // indirect
)

// Wait upstream fix: https://github.com/gdamore/tcell/issues/200
replace github.com/gdamore/tcell v1.1.1 => github.com/soyking/tcell v1.0.1-0.20180627092845-9addd5bbe425
