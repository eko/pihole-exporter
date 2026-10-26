package main

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

// setupLogging sets the log level and sends warnings and errors to stderr and
// the other logs to stdout. debug forces the debug level, as before LOG_LEVEL existed.
func setupLogging(
	level string,
	debug bool,
) error {
	if debug {
		level = "debug"
	}
	parsedLevel, err := log.ParseLevel(level)
	if err != nil {
		return err
	}

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetLevel(parsedLevel)
	log.SetOutput(io.Discard)
	log.AddHook(&writerHook{
		writer: os.Stderr,
		levels: []log.Level{log.PanicLevel, log.FatalLevel, log.ErrorLevel, log.WarnLevel},
	})
	log.AddHook(&writerHook{
		writer: os.Stdout,
		levels: []log.Level{log.InfoLevel, log.DebugLevel, log.TraceLevel},
	})
	return nil
}

// writerHook writes the log entries of the given levels to writer.
type writerHook struct {
	writer io.Writer
	levels []log.Level
}

func (h *writerHook) Levels() []log.Level {
	return h.levels
}

func (h *writerHook) Fire(entry *log.Entry) error {
	line, err := entry.Bytes()
	if err != nil {
		return err
	}
	_, err = h.writer.Write(line)
	return err
}
