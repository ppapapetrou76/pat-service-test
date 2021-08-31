package logstash

import (
	"encoding/json"
	"fmt"
	"time"

	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

type LogstashJsonFormatter struct {
	Env     string
	Service string
}

func (f *LogstashJsonFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields, len(entry.Data)+3)
	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/Sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}

	var pc uintptr
	var fileName string
	var lineNumber int
	for i := 2; i < 9; i++ {
		pc, fileName, lineNumber, _ = runtime.Caller(i)
		// If we need to debug the callstack, add this line
		//data[fmt.Sprintf("caller_%d", i)] = fmt.Sprintf("%s:%d", fileName, lineNumber)
		if !strings.Contains(fileName, "sirupsen") {
			break
		}
	}
	functionName := runtime.FuncForPC(pc).Name()

	data["@timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	data["hostname"] = os.Getenv("HOSTNAME")
	data["message"] = entry.Message
	data["logger"] = cleanUpFunctionName(functionName)
	data["lineno"] = fmt.Sprintf("%s:%d", fileName, lineNumber)
	data["level"] = entry.Level.String()
	data["env"] = f.Env
	data["service"] = f.Service

	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal fields to JSON, %v", err)
	}
	return append(serialized, '\n'), nil
}

func cleanUpFunctionName(name string) string {
	// function names are in the form of
	//github.com/sliide/hungrypenguin/services/httpd.Authenticate.func1
	//github.com/sliide/hungrypenguin/services/httpd.VerificationResultEvent
	//for a while we kept only the hungrypenguin/services/httpd.VerificationResultEvent part
	//the hungrypenguin/services/ is useless info and was never helpful in over a year
	//All searches happen on the function name itself
	//besides, the lineno is there in the log message

	// take just the last part of the / split
	split := strings.Split(name, "/")
	return split[len(split)-1]
}
