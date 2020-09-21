package log

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync/atomic"
)

const (
	Ldate         = log.Ldate
	Llongfile     = log.Llongfile
	Lmicroseconds = log.Lmicroseconds
	Lshortfile    = log.Lshortfile
	LstdFlags     = log.LstdFlags
	Ltime         = log.Ltime
)

type (
	LogType  int64
	LogLevel int64
)

const (
	TYPE_ERROR = LogType(1 << iota)
	TYPE_WARN
	TYPE_INFO
	TYPE_DEBUG
	TYPE_PANIC = LogType(^0)
)

const (
	LEVEL_NONE = LogLevel(1<<iota - 1)
	LEVEL_ERROR
	LEVEL_WARN
	LEVEL_INFO
	LEVEL_DEBUG
	LEVEL_ALL = LEVEL_DEBUG
)

func (t LogType) String() string {
	switch t {
	default:
		return "\t[LOG]"
	case TYPE_PANIC:
		return "\t[PANIC]"
	case TYPE_ERROR:
		return "\t[ERROR]"
	case TYPE_WARN:
		return "\t[WARN]"
	case TYPE_INFO:
		return "\t[INFO]"
	case TYPE_DEBUG:
		return "\t[DEBUG]"
	}
}

func String2LogLevel(str string) LogLevel {
	level := strings.ToLower(str)
	if level == "debug" {
		return LEVEL_DEBUG
	}

	if level == "info" {
		return LEVEL_INFO
	}

	if level == "warn" {
		return LEVEL_WARN
	}

	if level == "error" {
		return LEVEL_ERROR
	}

	if level == "none" {
		return LEVEL_NONE
	}

	Errorf("Unknown log level: %s, default level LEVEL_DEBUG will be used", str)
	return LEVEL_DEBUG
}

func (l *LogLevel) Set(v LogLevel) {
	atomic.StoreInt64((*int64)(l), int64(v))
}

func (l *LogLevel) Test(m LogType) bool {
	v := atomic.LoadInt64((*int64)(l))
	return (v & int64(m)) != 0
}

type nopCloser struct {
	io.Writer
}

func (*nopCloser) Close() error {
	return nil
}

func NopCloser(w io.Writer) io.WriteCloser {
	return &nopCloser{w}
}

type Logger struct {
	out   io.WriteCloser
	log   *log.Logger
	level LogLevel
}

var StdLog = New(NopCloser(os.Stdout), "", LEVEL_DEBUG)

func New(writer io.Writer, prefix string, level LogLevel) *Logger {
	out, ok := writer.(io.WriteCloser)
	if !ok {
		out = NopCloser(writer)
	}
	return &Logger{
		out:   out,
		log:   log.New(out, prefix, LstdFlags),
		level: level,
	}
}

func OpenFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	return f, err
}

func MustOpenFile(path string) *os.File {
	f, err := OpenFile(path)
	if err != nil {
		Panicf("open file log '%s' failed with error: %s", path, err)
	}
	return f
}

func FileLog(path string, logLevel string) (*Logger, error) {
	f, err := OpenFile(path)
	if err != nil {
		return nil, err
	}
	return New(f, "", String2LogLevel(logLevel)), nil
}

func MustFileLog(path string, logLevel string) *Logger {
	return New(MustOpenFile(path), "", String2LogLevel(logLevel))
}

func MustRollingLog(dir string, maxFileFrag int, maxFragSize int64, logLevel string) {
	_, procName := path.Split(os.Args[0])

	var path = dir + "/" + procName
	f, err := NewRollingFile(path, maxFileFrag, maxFragSize)
	if err != nil {
		Panicf("open rolling log file failed: %s, %s", path, err)
	} else {
		StdLog = New(f, "", String2LogLevel(logLevel))
	}

	SetFlags(Flags() | Lshortfile)
}

func (l *Logger) Flags() int {
	return l.log.Flags()
}

func (l *Logger) Prefix() string {
	return l.log.Prefix()
}

func (l *Logger) SetFlags(flags int) {
	l.log.SetFlags(flags)
}

func (l *Logger) SetPrefix(prefix string) {
	l.log.SetPrefix(prefix)
}

func (l *Logger) SetLevel(v LogLevel) {
	l.level.Set(v)
}

func (l *Logger) Close() {
	l.out.Close()
}

func (l *Logger) isDisabled(t LogType) bool {
	return t != TYPE_PANIC && !l.level.Test(t)
}

func (l *Logger) Panic(v ...interface{}) {
	t := TYPE_PANIC
	s := fmt.Sprint(v...)
	l.output(1, t, s)
	os.Exit(1)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	t := TYPE_PANIC
	s := fmt.Sprintf(format, v...)
	l.output(1, t, s)
	os.Exit(1)
}

func (l *Logger) Error(v ...interface{}) {
	t := TYPE_ERROR
	if l.isDisabled(t) {
		return
	}
	s := fmt.Sprint(v...)
	l.output(1, t, s)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	t := TYPE_ERROR
	if l.isDisabled(t) {
		return
	}
	s := fmt.Sprintf(format, v...)
	l.output(1, t, s)
}

func (l *Logger) Warn(v ...interface{}) {
	t := TYPE_WARN
	if l.isDisabled(t) {
		return
	}
	s := fmt.Sprint(v...)
	l.output(1, t, s)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	t := TYPE_WARN
	if l.isDisabled(t) {
		return
	}
	s := fmt.Sprintf(format, v...)
	l.output(1, t, s)
}

func (l *Logger) Info(v ...interface{}) {
	t := TYPE_INFO
	if l.isDisabled(t) {
		return
	}
	s := fmt.Sprint(v...)
	l.output(1, t, s)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	t := TYPE_INFO
	if l.isDisabled(t) {
		return
	}
	s := fmt.Sprintf(format, v...)
	l.output(1, t, s)
}

func (l *Logger) Debug(v ...interface{}) {
	t := TYPE_DEBUG
	if l.isDisabled(t) {
		return
	}
	s := fmt.Sprint(v...)
	l.output(1, t, s)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	t := TYPE_DEBUG
	if l.isDisabled(t) {
		return
	}
	s := fmt.Sprintf(format, v...)
	l.output(1, t, s)
}

func (l *Logger) Print(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.output(1, 0, s)
}

func (l *Logger) Printf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.output(1, 0, s)
}

func (l *Logger) Println(v ...interface{}) {
	s := fmt.Sprintln(v...)
	l.output(1, 0, s)
}

func (l *Logger) output(traceskip int, t LogType, s string) error {
	var b bytes.Buffer
	fmt.Fprint(&b, t, s)

	if len(s) == 0 || s[len(s)-1] != '\n' {
		fmt.Fprint(&b, "\n")
	}

	//fmt.Println(b.String())
	//return nil
	return l.log.Output(traceskip+2, b.String())
}

func Flags() int {
	return StdLog.log.Flags()
}

func Prefix() string {
	return StdLog.log.Prefix()
}

func SetFlags(flags int) {
	StdLog.log.SetFlags(flags)
}

func SetPrefix(prefix string) {
	StdLog.log.SetPrefix(prefix)
}

func SetLevel(v LogLevel) {
	StdLog.level.Set(v)
}

func Panic(v ...interface{}) {
	t := TYPE_PANIC
	s := fmt.Sprint(v...)
	StdLog.output(1, t, s)
	os.Exit(1)
}

func Panicf(format string, v ...interface{}) {
	t := TYPE_PANIC
	s := fmt.Sprintf(format, v...)
	StdLog.output(1, t, s)
	os.Exit(1)
}

func Error(v ...interface{}) {
	t := TYPE_ERROR
	if StdLog.isDisabled(t) {
		return
	}
	s := fmt.Sprint(v...)
	StdLog.output(1, t, s)
}

func Errorf(format string, v ...interface{}) {
	t := TYPE_ERROR
	if StdLog.isDisabled(t) {
		return
	}
	s := fmt.Sprintf(format, v...)
	StdLog.output(1, t, s)
}

func Warn(v ...interface{}) {
	t := TYPE_WARN
	if StdLog.isDisabled(t) {
		return
	}
	s := fmt.Sprint(v...)
	StdLog.output(1, t, s)
}

func Warnf(format string, v ...interface{}) {
	t := TYPE_WARN
	if StdLog.isDisabled(t) {
		return
	}
	s := fmt.Sprintf(format, v...)
	StdLog.output(1, t, s)
}

func Info(v ...interface{}) {
	t := TYPE_INFO
	if StdLog.isDisabled(t) {
		return
	}
	s := fmt.Sprint(v...)
	StdLog.output(1, t, s)
}

func Infof(format string, v ...interface{}) {
	t := TYPE_INFO
	if StdLog.isDisabled(t) {
		return
	}
	s := fmt.Sprintf(format, v...)
	StdLog.output(1, t, s)
}

func Debug(v ...interface{}) {
	t := TYPE_DEBUG
	if StdLog.isDisabled(t) {
		return
	}
	s := fmt.Sprint(v...)
	StdLog.output(1, t, s)
}

func Debugf(format string, v ...interface{}) {
	t := TYPE_DEBUG
	if StdLog.isDisabled(t) {
		return
	}
	s := fmt.Sprintf(format, v...)
	StdLog.output(1, t, s)
}

func Print(v ...interface{}) {
	s := fmt.Sprint(v...)
	StdLog.output(1, 0, s)
}

func Printf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	StdLog.output(1, 0, s)
}

func Println(v ...interface{}) {
	s := fmt.Sprintln(v...)
	StdLog.output(1, 0, s)
}
