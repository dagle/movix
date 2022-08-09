package movix

import (
	"fmt"
	"log"
	"os"
)

func LogInit(path string) error {
	logfile, err :=  os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	log.SetOutput(logfile)
	return nil
}

func Eprintf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
} 

func Fatal(v ...any) {
	fmt.Fprintf(os.Stderr, "%v\n", v...)
	os.Exit(1)
}

func Log(format string, a ...any) {
	log.Printf(format, a...)
}

func Elog(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
	log.Printf(format, a...)
}
