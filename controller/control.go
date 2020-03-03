package controller

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/poldi1405/BackUploader/display"
	"golang.org/x/sync/semaphore"
)

var (
	// Context
	Contxt = context.TODO()
	// Paths
	Path     = ""
	SuccPath = ""
	FailPath = ""
	// Commands
	Executor = ""
	ExecOpt  = ""
	PackCmd  = ""
	ParCmd   = ""
	UpldCmd  = ""
	// Semaphores
	Running   = semaphore.NewWeighted(0)
	Packing   = semaphore.NewWeighted(0)
	Paring    = semaphore.NewWeighted(0)
	Uploading = semaphore.NewWeighted(0)
	// additional Parameters
	PwdLength      = 5
	LogToFile      = ""
	DebugEnabled   = false
	LogFileHandler = os.Stderr
)

func Initialize() {
	if DebugEnabled {
		if LogToFile != "" {
			lfh, err := os.OpenFile(LogToFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
			if err != nil {
				panic(err)
			}
			LogFileHandler = lfh
		}
		log.SetOutput(LogFileHandler)
	}
}

const PATH_SEPARATOR = string(os.PathSeparator)

func Start(folder string, displayId int, wg *sync.WaitGroup) {
	defer wg.Done()
	defer Running.Release(1)
	cpath, err := filepath.Abs(Path + PATH_SEPARATOR + folder)
	if err != nil {
		display.Update(displayId, "@{!r}FAILED1!")
		return
	}
	replacevalues := [4]string{cpath + PATH_SEPARATOR + ".up",
		GenPwd(PwdLength),
		folder,
		SuccPath + PATH_SEPARATOR}
	// move out
	if err := os.Rename(cpath, Path+PATH_SEPARATOR+"._"+folder); err != nil {
		failed(cpath, folder, displayId, err)
		return
	}
	// recreate folder
	if err := os.Mkdir(cpath, os.ModePerm); err != nil {
		failed(cpath, folder, displayId, err)
	}
	// create upload folder
	if err := os.Mkdir(cpath+PATH_SEPARATOR+".up", os.ModePerm); err != nil {
		failed(cpath, folder, displayId, err)
	}
	// move to temporary folder
	if err := os.Rename(Path+PATH_SEPARATOR+"._"+folder, cpath+PATH_SEPARATOR+".tmp"); err != nil {
		failed(cpath, folder, displayId, err)
		return
	}

	// start packing
	Packing.Acquire(Contxt, 1)
	display.Update(displayId, "packing")
	if packing(cpath, replacevalues) {
		display.Update(displayId, "idle")
	} else {
		failed(cpath, folder, displayId, nil)
		return
	}

	// start creating parity
	Paring.Acquire(Contxt, 1)
	display.Update(displayId, "creating parity")
	if paring(cpath, replacevalues) {
		display.Update(displayId, "idle")
	} else {
		failed(cpath, folder, displayId, nil)
		return
	}

	// start uploading files
	Uploading.Acquire(Contxt, 1)
	display.Update(displayId, "uploading")
	if uploading(cpath, replacevalues) {
		display.Update(displayId, "@{!g}FINISHED!")
	} else {
		failed(cpath, folder, displayId, nil)
	}

	if err := os.Rename(cpath+PATH_SEPARATOR+".tmp", SuccPath+PATH_SEPARATOR+folder); err != nil {
		display.Update(displayId, "@{!y}UNABLE TO MOVE TO SUCCESS DIRECTORY!")
	}

	if err := os.RemoveAll(cpath); err != nil {
		display.Update(displayId, "@{!y}UNABLE TO CLEAN UP DIRECTORY!")
	}
}

func packing(folder string, values [4]string) bool {
	defer Packing.Release(1)
	packcommand := replace(PackCmd, values)
	if DebugEnabled {
		log.Println("Executing Command: ", Executor, ExecOpt, packcommand, "\n\tCurrent Working Directory: ", folder)
	}
	cmd := exec.Command(Executor, ExecOpt, packcommand)
	cmd.Dir = folder + PATH_SEPARATOR + ".tmp"

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if _, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if DebugEnabled {
					log.Println("Command: ", Executor, ExecOpt, packcommand, "\n\tin: ", folder, "failed")
				}
				return false
			}
		} else {
			log.Fatal(err)
		}
	}

	if DebugEnabled {
		log.Println("Command: ", Executor, ExecOpt, packcommand, "\n\tin: ", folder, "successful")
	}
	return true
}

func paring(folder string, values [4]string) bool {
	defer Paring.Release(1)
	packcommand := replace(ParCmd, values)
	if DebugEnabled {
		log.Println("Executing Command: ", Executor, ExecOpt, packcommand, "\n\tCurrent Working Directory: ", folder)
	}
	cmd := exec.Command(Executor, ExecOpt, packcommand)
	cmd.Dir = folder + PATH_SEPARATOR + ".up"

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if _, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if DebugEnabled {
					log.Println("Command: ", Executor, ExecOpt, packcommand, "\n\tin: ", folder, "failed")
				}
				return false
			}
		} else {
			log.Fatal(err)
		}
	}

	if DebugEnabled {
		log.Println("Command: ", Executor, ExecOpt, packcommand, "\n\tin: ", folder, "successful")
	}
	return true
}

func uploading(folder string, values [4]string) bool {
	defer Uploading.Release(1)
	packcommand := replace(UpldCmd, values)
	if DebugEnabled {
		log.Println("Executing Command: ", Executor, ExecOpt, packcommand, "\n\tCurrent Working Directory: ", folder)
	}
	cmd := exec.Command(Executor, ExecOpt, packcommand)
	cmd.Dir = folder + PATH_SEPARATOR + ".up"

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if _, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if DebugEnabled {
					log.Println("Command: ", Executor, ExecOpt, packcommand, "\n\tin: ", folder, "failed")
				}
				return false
			}
		} else {
			log.Fatal(err)
		}
	}

	if DebugEnabled {
		log.Println("Command: ", Executor, ExecOpt, packcommand, "\n\tin: ", folder, "successful")
	}
	return true
}

func failed(path string, folder string, displayId int, err error) {
	display.Update(displayId, "@{!r}FAILED!")
	if err != nil {
		log.Print(err)
	}
	if err := os.Rename(path, FailPath+PATH_SEPARATOR+folder); err != nil {
		display.Update(displayId, "@{!r^}UNABLE TO MOVE TO FAILED FOLDER!")
		log.Fatal(err)
	}
}

func replace(str string, values [4]string) string {
	for key, value := range values {
		str = strings.Replace(str, "{"+strconv.Itoa(key)+"}", value, -1)
	}
	return str
}
