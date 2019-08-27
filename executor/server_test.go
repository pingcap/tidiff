package executor

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var defaultMySQLDSNConfig = &Config{
	User:    "root",
	Host:    "localhost",
	Port:    3306,
	DB:      "test",
	Options: "charset=utf8mb4&collation=utf8mb4_bin",
}

var defaultTiDBDSNConfig = &Config{
	User:    "root",
	Host:    "127.0.0.1",
	Port:    4000,
	DB:      "test",
	Options: "charset=utf8mb4&collation=utf8mb4_bin",
}

func Example() {
	exec := NewExecutor(defaultMySQLDSNConfig, defaultTiDBDSNConfig)
	if err := exec.Open(DefaultRetryCnt); err != nil {
		fmt.Printf("open failed err %v", err)
		return
	}

	sqls := []string{"drop table t", "create table t(a int)", "show create table t", "drop table t"}
	if err := exec.DoDiff(sqls, nil); err != nil {
		fmt.Printf("do diff failed \n%v", err)
	}
}
