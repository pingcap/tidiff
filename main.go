package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
	"github.com/lonng/tidiff/executor"
	"github.com/lonng/tidiff/history"
	"github.com/lonng/tidiff/ui"
	"github.com/sergi/go-diff/diffmatchpatch"
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
			Value: "test",
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
			Value: "test",
			Usage: "TiDB database",
		},
		cli.StringFlag{
			Name:  "tidb.options",
			Value: "charset=utf8mb4",
			Usage: "TiDB DSN options",
		},
		cli.StringFlag{
			Name:  "log.diff",
			Value: "",
			Usage: "Log all query diff to file",
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
	mysql, err := sql.Open("mysql", dsn("mysql", ctx))
	if err != nil {
		return err
	}
	tidb, err := sql.Open("mysql", dsn("tidb", ctx))
	if err != nil {
		return err
	}
	exec := executor.NewExecutor(mysql, tidb)

	// Command line mode
	if args := ctx.Args(); len(args) > 0 {
		query := strings.Join(args, " ")
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
		fmt.Println("MySQL> " + mysqlResult.Rendered)
		fmt.Println(mysqlContent)
		fmt.Println(mysqlResult.Stat() + "\n")
		fmt.Println("TiDB> " + tidbResult.Rendered)
		fmt.Println(tidbContent)
		fmt.Println(tidbResult.Stat() + "\n")
		return nil
	}

	// User interface mode
	recorder := history.NewRecorder()
	if err := recorder.Open(); err != nil {
		return err
	}
	defer func() {
		if err := recorder.Close(); err != nil {
			log.Println(err.Error())
		}
	}()

	if logDiff := ctx.String("log.diff"); logDiff != "" {
		diff, err := os.OpenFile(logDiff, os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
		recorder.SetDiff(diff)
		defer diff.Close()
	}

	return ui.New(recorder, exec, mysql, tidb).Serve()
}
