package utils

import (
	"log"
	"os"
	"syscall"

	"github.com/kardianos/osext"
)

func Restart() {
	file, err := osext.Executable()
	if err != nil {
		log.Println("restart:", err)
	} else {
		err = syscall.Exec(file, os.Args, os.Environ())
		if err != nil {
			log.Fatal(err)
		}
	}
}
