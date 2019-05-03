package executor

import "fmt"

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DB       string
	Options  string
}

func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", c.User, c.Password, c.Host, c.Port, c.DB, c.Options)
}

func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
