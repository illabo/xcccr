package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	fmt.Println("Xcode Coverage for Commit Reporter (https://github.com/illabo/xcccr)")

	rnCnd, err := prepareRunConditions()
	if err != nil {
		log.Fatal(err)
	}

	performReport(rnCnd)
}

func getWorkdir() string {
	dir, _ := os.Getwd()
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	if dir == "" {
		dir = "~/"
	}
	return appendSlash(dir)
}

func appendSlash(pathStr string) string {
	if strings.HasSuffix(pathStr, "/") {
		return pathStr
	}
	return pathStr + "/"
}

func randomStr(ln int) string {
	set := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rb := make([]byte, ln)
	for i := range rb {
		rb[i] = set[rand.Intn(len(set))]
	}
	return string(rb)
}
