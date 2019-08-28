package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pingcap/tidiff/config"
	"github.com/pingcap/tidiff/executor"
	"github.com/pingcap/tidiff/history"
	"github.com/pingcap/tidiff/uimode"
	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/urfave/cli.v2"
)

func main() {
	app := cli.App{}
	app.Name = "tidiff"
	app.Usage = "Execute SQL in TiDB and MySQL and returns the results"
	app.Description = "Used to compare the result different in MySQL and TiDB for the same SQL statement"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "mysql.host",
			Value: "127.0.0.1",
			Usage: "MySQL host",
		},
		&cli.IntFlag{
			Name:  "mysql.port",
			Value: 3306,
			Usage: "MySQL port",
		},
		&cli.StringFlag{
			Name:  "mysql.user",
			Value: "root",
			Usage: "MySQL username",
		},
		&cli.StringFlag{
			Name:  "mysql.password",
			Value: "",
			Usage: "MySQL password",
		},
		&cli.StringFlag{
			Name:  "mysql.db",
			Value: "",
			Usage: "MySQL database",
		},
		&cli.StringFlag{
			Name:  "mysql.options",
			Value: "charset=utf8mb4",
			Usage: "MySQL DSN options",
		},
		&cli.StringFlag{
			Name:  "tidb.host",
			Value: "127.0.0.1",
			Usage: "TiDB host",
		},
		&cli.IntFlag{
			Name:  "tidb.port",
			Value: 4000,
			Usage: "TiDB port",
		},
		&cli.StringFlag{
			Name:  "tidb.user",
			Value: "root",
			Usage: "TiDB username",
		},
		&cli.StringFlag{
			Name:  "tidb.password",
			Value: "",
			Usage: "TiDB password",
		},
		&cli.StringFlag{
			Name:  "tidb.db",
			Value: "",
			Usage: "TiDB database",
		},
		&cli.StringFlag{
			Name:  "tidb.options",
			Value: "charset=utf8mb4",
			Usage: "TiDB DSN options",
		},
		&cli.StringFlag{
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

func dbConfig(dialect string, ctx *cli.Context) *executor.Config {
	return &executor.Config{
		Host:     ctx.String(dialect + ".host"),
		Port:     ctx.Int(dialect + ".port"),
		User:     ctx.String(dialect + ".user"),
		Password: ctx.String(dialect + ".password"),
		DB:       ctx.String(dialect + ".db"),
		Options:  ctx.String(dialect + ".options"),
	}
}

func initConfig(ctx *cli.Context) error {
	err := os.MkdirAll(filepath.Join(config.TiDiffPath), os.ModePerm)
	if err != nil {
		return err
	}
	_, err = os.Stat(config.TiDiffConfigPath)
	if err != nil && os.IsNotExist(err) {
		file, err := os.Create(config.TiDiffConfigPath)
		if err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
	}

	b, err := ioutil.ReadFile(config.TiDiffConfigPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		parts[0] = strings.TrimSpace(parts[0])
		parts[1] = strings.TrimSpace(parts[1])
		if ctx.IsSet(parts[0]) {
			continue
		}
		if err := ctx.Set(parts[0], parts[1]); err != nil {
			return err
		}
	}
	return nil
}

func serveCLIMode(ctx *cli.Context, exec *executor.Executor) error {
	query := strings.Join(ctx.Args().Slice(), " ")
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
	logQuery := query
	if strings.HasPrefix(query, "!!") {
		logQuery = mysqlResult.Rendered
	}
	fmt.Println(fmt.Sprintf("MySQL(%s)> %s", exec.MySQLConfig.Address(), logQuery))
	if mysqlContent != "" {
		fmt.Println(mysqlContent)
	}
	fmt.Println(mysqlResult.Stat() + "\n")
	fmt.Println(fmt.Sprintf("TiDB(%s)> %s", exec.TiDBConfig.Address(), logQuery))
	if tidbContent != "" {
		fmt.Println(tidbContent)
	}
	fmt.Println(tidbResult.Stat() + "\n")
	return nil
}

func serve(ctx *cli.Context) error {
	if err := initConfig(ctx); err != nil {
		return err
	}
	exec := executor.NewExecutor(dbConfig("mysql", ctx), dbConfig("tidb", ctx))
	if err := exec.Open(executor.DefaultRetryCnt); err != nil {
		return err
	}

	// Command line mode
	if args := ctx.Args(); args.Len() > 0 {
		return serveCLIMode(ctx, exec)
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

	return uimode.New(recorder, exec).Serve()
}
