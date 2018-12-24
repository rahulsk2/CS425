package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	f, err := os.Open("ListOfPossibleNodes")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(),":")
		fmt.Println("\nIP:" + line[0])
		fmt.Println("Port:" + line[1])

	}
	err = scanner.Err()
	return
}