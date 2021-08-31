package logstash

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	LogLevels = map[string]log.Level{
		"DEBUG":   log.DebugLevel,
		"INFO":    log.InfoLevel,
		"WARNING": log.WarnLevel,
		"ERROR":   log.ErrorLevel,
		"FATAL":   log.FatalLevel,
		"PANIC":   log.PanicLevel,
	}

	rotationCheckInterval       = 10 * time.Second
	maxLogFileSize        int64 = 2 << 30 // 2GiB

	stopRotation chan struct{}
)

// Init sets up logging
// This function should only be called once when the service is started
func Init(logLevel string, logFileName string, env string, service string, maxSize int64) bool {

	if maxSize != 0 {
		maxLogFileSize = maxSize
	}

	logToStdout := os.Getenv("LOG_TO_STDOUT") == "1"

	log.SetFormatter(&LogstashJsonFormatter{
		Env:     env,
		Service: service,
	})

	log.SetLevel(LogLevels[logLevel])

	if logToStdout {
		setOutput(os.Stdout)
		return true
	}

	logFile, err := os.Create(logFileName)
	if err != nil {
		panic(err)
	}

	setOutput(logFile)
	go rotate(logFileName)
	return true
}

// InitWithOutput sets up logging with a given output
// This function should only be called once when the service is started
func InitWithOutput(logLevel, env, service string, output io.Writer) error {
	level, ok := LogLevels[logLevel]
	if !ok {
		return fmt.Errorf("Unsupported log level: %s", logLevel)
	}

	log.SetFormatter(&LogstashJsonFormatter{
		Env:     env,
		Service: service,
	})

	log.SetLevel(level)
	log.SetOutput(output)
	return nil
}

func setOutput(writer io.Writer) {
	log.SetOutput(writer)
}

func rotationTicker(logFileName string) {

}

// rotate checks periodically if the
// when the current logfile
func rotate(logFileName string) {
	log.Debug("starting rotation for", logFileName)

	if stopRotation != nil {
		log.Debug("stopping rotation")
		close(stopRotation)
	}
	stopRotation = make(chan struct{})

	ticker := time.NewTicker(rotationCheckInterval)
	for {
		select {
		case <-stopRotation:
			log.Debug("stopping rotation for", logFileName)
			ticker.Stop()
			return
		case <-ticker.C:
			f, err := os.Stat(logFileName)
			if f.Size() > maxLogFileSize {
				log.Debugf("log file too large, rotating the log file")
				rotatedFilename := logFileName + ".1"

				// if the rotated file already exist, add timestamp to its name and archive it
				if _, err = os.Stat(rotatedFilename); err == nil {

					err = delayedCompression(rotatedFilename)
					if err != nil {
						// if the renaming for compression failed, try again before rotating
						continue
					}
				}

				err = os.Rename(logFileName, rotatedFilename)
				if err != nil {
					log.Error("couldn't rename log file", logFileName, err)
					continue
				}

				NewLogFile, err := os.Create(logFileName)
				if err != nil {
					panic(err)
				}

				setOutput(NewLogFile)
			}
		}

	}
}

// Rename and compress old logs
func delayedCompression(rotatedFilename string) error {

	timestampedName := rotatedFilename + time.Now().Format(time.RFC3339)

	log.Debugf("found previous log file %s, renaming to %s", rotatedFilename, timestampedName)

	err := os.Rename(rotatedFilename, timestampedName)
	if err != nil {
		log.Error("couldn't rename rotated log file", rotatedFilename, err)
		return err
	}
	cmd := exec.Command("gzip", timestampedName)
	go func() {
		log.Debug("Compressing log file", timestampedName)
		err := cmd.Run()
		if err != nil {
			log.Error("gzip on rotated log file failed", timestampedName, err)
		}
	}()
	return nil
}
