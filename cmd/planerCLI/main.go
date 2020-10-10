package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/devsquared/planer/pkg/planer"
)

func main() {
	// TODO: typing a timestamp in a cli seems awkward, should give a yaml config to ease this; then look at a wails app
	// TODO: create multiple methods to allow for word search in log
	// TODO: provide a file structure to drop in log files and get output files for saving
	// TODO: provide way to define the time format
	// TODO: look at merging with the file search project
	if err := cli(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func cli() error {
	// setup a ctrl-c interrupt to cancel and exit the program
	cancelled := setupCloseHandler()

	fmt.Println("Welcome to Planer, the easy way to make your logs manageable.")
	fmt.Println("-------------------------------------------------------------")

	// TODO: when we implement term filtering, we will ask if the user wants this
	// and what the term is

	// prompt a user for the args we need
	fmt.Println("To filter down the log, what is the 'from' portion of the time range?")
	fmt.Println("(Format the timestamp as such - 2006-01-02T15:04:05.0000Z")

	fromString, err := getInputText()
	if err != nil {
		fmt.Println("Issue getting from timestamp input text.")
		return err
	}

	from, err := time.Parse("2006-01-02T15:04:05.0000Z", fromString)
	if err != nil {
		fmt.Println("Unable to parse the from time arg : ", fromString)
		return err
	}

	fmt.Println("What is the 'to' portion of the time range? Alternatively, leave arg blank to default to now.")
	fmt.Println("(Format the timestamp as such - 2006-01-02T15:04:05.0000Z")

	toString, err := getInputText()
	if err != nil {
		fmt.Println("Issue getting the to timestamp input text.")
		return err
	}

	var to time.Time
	if toString == "" {
		to = time.Now()
	} else {
		to, err = time.Parse("2006-01-02T15:04:05.0000Z", toString)
		if err != nil {
			fmt.Println("Unable to parse the to time arg : ", toString)
			return err
		}
	}

	fmt.Println("What is the log file to plane down?")
	fmt.Println("Please provide a full path.")

	fileName, err := getInputText()
	if err != nil {
		fmt.Println("Issue getting the fileName input text.")
		return err
	}

	fmt.Println("What is the word to filter by?")
	fmt.Println("(If you have no filter word, leave this blank.)")

	filterWord, err := getInputText()
	if err != nil {
		fmt.Println("Issue getting the filter word.")
		return err
	}

	if !cancelled {
		message, err := planer.PlaneLog(from, to, filterWord, fileName)
		if err != nil {
			return err
		}

		fmt.Println(message)
	}

	return nil
}

func getInputText() (string, error) {
	// prompt for input
	fmt.Print(">> ")

	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	text = strings.TrimSuffix(text, "\n")

	return text, nil
}

// NOTE: This close handler does not seem to work in situations where a IDE handles the running of the app
// It, however, works as intended in a terminal.
func setupCloseHandler() bool {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() bool {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		os.Exit(0)
		return true
	}()

	//base case
	return false
}
