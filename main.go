package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/bigkevmcd/go-configparser"
	"github.com/poldi1405/BackUploader/controller"
	"github.com/poldi1405/BackUploader/display"
	"github.com/wsxiaoys/terminal/color"
	"golang.org/x/sync/semaphore"
)

var (
	Wg           sync.WaitGroup
	buildVersion = "unknown! This was not built using the Makefile!"
	buildArch    = "unknown!"
	setulimit    = int64(10)
	setclimit    = int64(10)
	setprlimit   = int64(10)
	setpclimit   = int64(10)
	setpwdlength = int64(5)
	conf         *configparser.ConfigParser
	conferr      error
)

func init() {
	conf, conferr = configparser.NewConfigParserFromFile("config.ini")
}

func commandLineParser() {
	if len(os.Args) > 1 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			fmt.Println("BackUploader\n")
			fmt.Println("-h --help\tThis Helpmessage")
			fmt.Println("-v\tPrint version and exit")
			fmt.Println("-d [file]\tEnable debug-mode and log to file")
			os.Exit(0)
		}
		if os.Args[1] == "-v" {
			fmt.Println("Backuploader Version", buildVersion)
			if buildArch == "-" {
				buildArch = "equal to compiling system"
			}
			fmt.Println("Architecture:", buildArch)
			os.Exit(0)
		}
		if os.Args[1] == "-d" {
			controller.DebugEnabled = true
			if len(os.Args) == 3 {
				controller.LogToFile = os.Args[2]
			}
		}
	}
}

func configFileParser() {
	// get limit for concurrent jobs
	setclimitc, err := conf.GetInt64("Settings", "Concurrent")
	if err != nil {
		fmt.Println("no concurrency limit set. using default (10).")
	} else {
		setclimit = setclimitc
	}
	// get limit for concurrent packing
	setpclimitc, err := conf.GetInt64("Settings", "Packing")
	if err != nil {
		fmt.Println("no packing limit set. using default (2).")
	} else {
		setpclimit = setpclimitc
	}
	// get limit for concurrent paring
	setprlimitc, err := conf.GetInt64("Settings", "Paring")
	if err != nil {
		fmt.Println("no paring limit set. using default (3).")
	} else {
		setprlimit = setprlimitc
	}
	// get limit for concurrent uploading
	setulimitc, err := conf.GetInt64("Settings", "Uploading")
	if err != nil {
		fmt.Println("no upload limit set. using default (5).")
	} else {
		setulimit = setulimitc
	}

	// get password_length
	setpwdlength, err = conf.GetInt64("Settings", "PasswordLength")
	if err != nil {
		setpwdlength = 16
	}

	// get commands...
	// ...for packing
	controller.PackCmd, err = conf.Get("Commands", "Packing")
	if err != nil {
		fmt.Println("Unable to get command for packing! Aborting...")
		os.Exit(1)
	}
	// ...for paring
	controller.ParCmd, err = conf.Get("Commands", "Paring")
	if err != nil {
		fmt.Println("Unable to get command for paring! Aborting...")
		os.Exit(1)
	}
	// ...for uploading
	controller.UpldCmd, err = conf.Get("Commands", "Uploading")
	if err != nil {
		fmt.Println("Unable to get command for uploading! Aborting...")
		os.Exit(1)
	}
	// ...executed...
	controller.Executor, err = conf.Get("Commands", "Executor")
	if err != nil {
		fmt.Println("Unable to get terminal to execute in! Aborting...")
		os.Exit(1)
	}
	// ...and its options
	controller.ExecOpt, err = conf.Get("Commands", "ExecOpt")
	if err != nil {
		fmt.Println("No ExecOpt has been set! Most Terminals require a special Option to execute external Commands. If the execution fails, please check the documentation of your terminal for additional information.")
	}
	// get folders
	// ...for uploads
	controller.Path, err = conf.Get("Directories", "Upload")
	if err != nil {
		fmt.Println("Unable to get command for uploading! Aborting...")
		os.Exit(1)
	}
	// ...for uploading
	controller.FailPath, err = conf.Get("Directories", "Failed")
	if err != nil {
		fmt.Println("Unable to get command for uploading! Aborting...")
		os.Exit(1)
	}
	// ...for uploading
	controller.SuccPath, err = conf.Get("Directories", "Finished")
	if err != nil {
		fmt.Println("Unable to get command for uploading! Aborting...")
		os.Exit(1)
	}
}

func main() {
	// parse Commandline
	commandLineParser()
	controller.Initialize()

	// read and parse configuration
	fmt.Print("Reading Config........")
	if conferr != nil {
		color.Println("@{^r*!}ERROR")
		panic(conferr)
	}

	configFileParser()

	color.Println("@{g}DONE@{|}")

	// read value from config
	fmt.Print("Listing Directories...")
	path, err := conf.Get("Directories", "Upload")
	if err != nil {
		color.Println("@{^r*!}ERROR")
		return
	}
	// try to read subdirectories
	subdirs, err := ioutil.ReadDir(path)
	if err != nil {
		color.Println("@{^r*!}ERROR")
		return
	}
	color.Println("@{g}DONE@{|}")

	var dirs = make([]string, 0)
	for _, dir := range subdirs {
		if dir.IsDir() {
			dirs = append(dirs, dir.Name())
		}
	}

	color.Println("Found a total of@{!}", len(dirs), "@{|}directories.")
	if len(dirs) == 0 {
		fmt.Println("Nothing to do.")
		return
	}

	//TODO: Context controller
	fmt.Println("a maximum of", setclimit, "jobs is performed at a time")
	controller.Running = semaphore.NewWeighted(setclimit)
	fmt.Println("a maximum of", setpclimit, "jobs is packed at a time")
	controller.Packing = semaphore.NewWeighted(setpclimit)
	fmt.Println("a maximum of", setprlimit, "jobs is pared at a time")
	controller.Paring = semaphore.NewWeighted(setprlimit)
	fmt.Println("a maximum of", setulimit, "jobs is uploaded at a time")
	controller.Uploading = semaphore.NewWeighted(setulimit)
	controller.PwdLength = int(setpwdlength)
	controller.Path = path

	for _, nextdir := range dirs {
		controller.Running.Acquire(controller.Contxt, 1)
		displayId := display.Add("@{y}idle", nextdir)
		go controller.Start(nextdir, displayId, &Wg)
		Wg.Add(1)
	}
	Wg.Wait()
	fmt.Println("Group completed")
}
