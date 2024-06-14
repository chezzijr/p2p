package peer

import (
	"io"
	"time"

	"github.com/charmbracelet/log"
)

var logger *log.Logger

func InitLogger(w io.Writer) {
    logger = log.NewWithOptions(w, log.Options{
        ReportCaller: true,
        ReportTimestamp: true,
        TimeFormat: time.Kitchen,
    })
}
