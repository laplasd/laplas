package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

func Init() {
	// Формат вывода — текстовый с цветами
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Вывод в stdout
	Log.SetOutput(os.Stdout)

	// Минимальный уровень логов — Info
	Log.SetLevel(logrus.DebugLevel)
	//Log.SetReportCaller(true)
}
