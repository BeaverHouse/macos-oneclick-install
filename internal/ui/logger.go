package ui

import "github.com/BeaverHouse/go-common/logger"

var Log logger.Logger

func init() {
	l, err := logger.NewLogger()
	if err != nil {
		panic(err)
	}
	Log = l
}
