package logger

import (
	"fmt"

	au "github.com/logrusorgru/aurora"
)

const (
	successMark = "[+]"
	failureMark = "[!]"
)

type Logger struct {
	au      au.Aurora
	verbose bool
}

func NewLogger(verbose, colors bool) *Logger {
	return &Logger{
		au:      au.NewAurora(colors),
		verbose: verbose,
	}
}

func (l *Logger) Log(msg string) {
	if l.verbose {
		m := successMark + " " + msg
		fmt.Println(l.au.Cyan(m))
	}
}

func (l *Logger) LogF(msg string, args ...any) {
	l.Log(fmt.Sprintf(msg, args...))
}

func (l *Logger) Error(msg string, err error) {
	if l.verbose {
		m := failureMark + " " + msg
		fmt.Print(l.au.Red(m))
		if err != nil {
			fmt.Print(l.au.Red(err))
		}
		fmt.Println()
	}
}
