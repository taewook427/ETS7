// test828 : ETS7 TestLog
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var FILE_NAME string = "testlog.txt"

func initEnv() {
	// move to executable path
	exePath, _ := os.Executable()
	realPath, _ := filepath.EvalSymlinks(exePath)
	os.Chdir(filepath.Dir(realPath))

	// make new record if not exists
	if _, err := os.Stat(FILE_NAME); os.IsNotExist(err) {
		iLog := fmt.Sprintf("0,%d,new log generated\n", time.Now().Unix())
		os.WriteFile(FILE_NAME, []byte(iLog), 0644)
	}
}

func input(q string) string {
	fmt.Print(q)
	temp, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if len(temp) == 0 {
		return ""
	} else if temp[len(temp)-1] == '\n' {
		temp = temp[0 : len(temp)-1]
	}
	if len(temp) == 0 {
		return ""
	} else if temp[len(temp)-1] == '\r' {
		temp = temp[0 : len(temp)-1]
	}
	return temp
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			os.WriteFile("panic-log.txt", fmt.Appendf(nil, "panic while testlog.main: %v", r), 0644)
		}
	}()
	initEnv()
	lastNum := -1

	// read existing log and print
	data, err := os.ReadFile(FILE_NAME)
	if err != nil {
		fmt.Printf("[ERROR] file open fail: %v", err)
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		// parse line
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ",", 3)
		if len(parts) < 3 {
			continue
		}

		// parse integer
		num, err0 := strconv.Atoi(parts[0])
		unixTime, err1 := strconv.ParseInt(parts[1], 10, 64)
		if err0 != nil || err1 != nil {
			continue
		}
		if num > lastNum {
			lastNum = num
		}

		// print record
		t := time.Unix(unixTime, 0)
		fmt.Printf("[%s] %6d: %s\n", t.Format("2006-01-02 15:04:05"), num, parts[2])
	}

	// open file to add record
	file, err := os.OpenFile(FILE_NAME, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[ERROR] file open fail: %v", err)
		return
	}
	defer file.Close()

	// input loop
	nextNum := lastNum + 1
	for {
		now := time.Now()
		prompt := fmt.Sprintf("[%s] %6d: ", now.Format("2006-01-02 15:04:05"), nextNum)
		text := input(prompt)
		if text == "" {
			break
		}

		// add record
		logLine := fmt.Sprintf("%d,%d,%s\n", nextNum, now.Unix(), text)
		if _, err := file.WriteString(logLine); err != nil {
			fmt.Printf("[ERROR] file write fail: %v", err)
			break
		}
		nextNum++
	}
}
