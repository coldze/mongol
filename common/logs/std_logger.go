package logs

import "log"

const (
	log_debug_prefix   = "[DEBUG] "
	log_info_prefix    = "[INFO] "
	log_warning_prefix = "[WARNING] "
	log_error_prefix   = "[ERROR] "
	log_fatal_prefix   = "[FATAL] "
)

type stdLogger struct {
}

func (l *stdLogger) Debugf(format string, args ...interface{}) {
	log.Printf(log_debug_prefix+format, args...)
}

func (l *stdLogger) Infof(format string, args ...interface{}) {
	log.Printf(log_info_prefix+format, args...)
}

func (l *stdLogger) Warningf(format string, args ...interface{}) {
	log.Printf(log_warning_prefix+format, args...)
}

func (l *stdLogger) Errorf(format string, args ...interface{}) {
	log.Printf(log_error_prefix+format, args...)
}

func (l *stdLogger) Fatalf(format string, args ...interface{}) {
	log.Printf(log_fatal_prefix+format, args...)
}

func NewStdLogger() Logger {
	return &stdLogger{}
}
