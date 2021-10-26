package consul

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	hclog "github.com/hashicorp/go-hclog"
	"go.unistack.org/micro/v3/logger"
)

// to check implementation
var _ hclog.Logger = &consulLogger{}

type consulLogger struct {
	logger logger.Logger
}

func (l *consulLogger) Name() string {
	return l.logger.String()
}

func (l *consulLogger) With(args ...interface{}) hclog.Logger {
	fields := make(map[string]interface{}, int(len(args)/2))
	for i := 0; i < int(len(args)/2); i += 2 {
		fields[fmt.Sprintf("%v", args[i])] = args[i+1]
	}
	return &consulLogger{logger: l.logger.Fields(fields)}
}

func (l *consulLogger) Debug(format string, msg ...interface{}) {
	l.logger.Debugf(context.TODO(), format, msg...)
}

func (l *consulLogger) Error(format string, msg ...interface{}) {
	l.logger.Errorf(context.TODO(), format, msg...)
}

func (l *consulLogger) Info(format string, msg ...interface{}) {
	l.logger.Infof(context.TODO(), format, msg...)
}

func (l *consulLogger) Warn(format string, msg ...interface{}) {
	l.logger.Warnf(context.TODO(), format, msg...)
}

func (l *consulLogger) Trace(format string, msg ...interface{}) {
	l.logger.Tracef(context.TODO(), format, msg...)
}

func (l *consulLogger) ImpliedArgs() []interface{} {
	fields := make([]interface{}, len(l.logger.Options().Fields)*2)
	for k, v := range l.logger.Options().Fields {
		fields = append(fields, k, v)
	}
	return fields
}

func (l *consulLogger) Named(name string) hclog.Logger {
	var newname string
	var oldname string

	fields := l.logger.Options().Fields
	for i := 0; i < len(fields); i += 2 {
		if fields[i].(string) == "name" {
			oldname = fields[i+1].(string)
		}
	}

	if len(oldname) > 0 {
		newname = fmt.Sprintf("%s.%s", oldname, name)
	} else {
		newname = name
	}
	return &consulLogger{logger: l.logger.Fields("name", newname)}
}

func (l *consulLogger) ResetNamed(name string) hclog.Logger {
	return &consulLogger{logger: l.logger.Fields("name", name)}
}

func (l *consulLogger) SetLevel(level hclog.Level) {
	// TODO: add logic when logger.Logger supports this method
}

func (l *consulLogger) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	// TODO: add logic
	return log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Llongfile|log.LUTC)
}

func (l *consulLogger) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	buf := bytes.NewBuffer(nil)
	return buf
}

func (l *consulLogger) IsDebug() bool {
	return l.logger.V(logger.DebugLevel)
}

func (l *consulLogger) IsError() bool {
	return l.logger.V(logger.ErrorLevel)
}

func (l *consulLogger) IsInfo() bool {
	return l.logger.V(logger.InfoLevel)
}

func (l *consulLogger) IsTrace() bool {
	return l.logger.V(logger.TraceLevel)
}

func (l *consulLogger) IsWarn() bool {
	return l.logger.V(logger.WarnLevel)
}

func (l *consulLogger) Log(level hclog.Level, msg string, args ...interface{}) {
	switch level {
	case hclog.Trace:
		l.Trace(msg, args...)
	case hclog.Debug:
		l.Debug(msg, args...)
	case hclog.Info:
		l.Info(msg, args...)
	case hclog.Warn:
		l.Warn(msg, args...)
	case hclog.Error:
		l.Error(msg, args...)
	case hclog.NoLevel:
		l.Info(msg, args...)
	}
}
