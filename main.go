package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/lonng/0x81/history"
	"github.com/lonng/0x81/ui"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "tidb"
	app.Usage = "Execute SQL in TiDB and MySQL and returns the results"
	app.Description = "Used to compare the result different in MySQL and TiDB for the same SQL statement"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "mysql.host",
			Value: "127.0.0.1",
			Usage: "MySQL host",
		},
		cli.IntFlag{
			Name:  "mysql.port",
			Value: 3306,
			Usage: "MySQL port",
		},
		cli.StringFlag{
			Name:  "mysql.user",
			Value: "root",
			Usage: "MySQL username",
		},
		cli.StringFlag{
			Name:  "mysql.password",
			Value: "",
			Usage: "MySQL password",
		},
		cli.StringFlag{
			Name:  "mysql.db",
			Value: "",
			Usage: "MySQL database",
		},
		cli.StringFlag{
			Name:  "mysql.options",
			Value: "charset=utf8mb4",
			Usage: "MySQL DSN options",
		},
		cli.StringFlag{
			Name:  "tidb.host",
			Value: "127.0.0.1",
			Usage: "TiDB host",
		},
		cli.IntFlag{
			Name:  "tidb.port",
			Value: 4000,
			Usage: "TiDB port",
		},
		cli.StringFlag{
			Name:  "tidb.user",
			Value: "root",
			Usage: "TiDB username",
		},
		cli.StringFlag{
			Name:  "tidb.password",
			Value: "",
			Usage: "TiDB password",
		},
		cli.StringFlag{
			Name:  "tidb.db",
			Value: "",
			Usage: "TiDB database",
		},
		cli.StringFlag{
			Name:  "tidb.options",
			Value: "charset=utf8mb4",
			Usage: "TiDB DSN options",
		},
	}
	app.Action = serve
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func dsn(dialect string, ctx *cli.Context) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		ctx.String(dialect+".user"),
		ctx.String(dialect+".password"),
		ctx.String(dialect+".host"),
		ctx.Int(dialect+".port"),
		ctx.String(dialect+".db"),
		ctx.String(dialect+".options"),
	)
}

func serve(ctx *cli.Context) error {
	recorder := history.NewRecorder()
	err := recorder.Open()
	if err != nil {
		return err
	}
	defer recorder.Close()

	mysql, err := sql.Open("mysql", dsn("mysql", ctx))
	if err != nil {
		return err
	}
	tidb, err := sql.Open("mysql", dsn("tidb", ctx))
	if err != nil {
		return err
	}

	return ui.New(recorder, mysql, tidb).Serve()
}
