package workers

import (
	"os"
	"testing"

	"github.com/ProgrammingLab/koneko-online-judge/server/logger"
	"github.com/labstack/gommon/log"
)

func TestMain(m *testing.M) {
	logger.AppLog = log.New("")

	os.Exit(m.Run())
}
