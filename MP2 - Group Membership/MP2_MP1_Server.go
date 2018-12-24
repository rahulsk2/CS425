package main

import (
	"fmt"
	"strings"
)
import "net"
import "bufio"
import "os/exec"
import "os"



func main() {
	checkArguments()

	fileName := "machine.log"
	port := os.Args[1]
	server,err := net.Listen("tcp",":"+port)
	errorHandler(err,"Could not start server!", true)
	for {
		conn, err := server.Accept()
		errorHandler(err, "Couldn't open connection", true)

		go func (conn net.Conn){
			defer conn.Close()
			input := bufio.NewReader(conn)
			output := bufio.NewWriter(conn)
			pattern,err := input.ReadString('\n')
			pattern=pattern[:len(pattern)-1]
			fmt.Println("Pattern:" + pattern)
			s := strings.Split(pattern, ":")
			query, flag := s[0], s[1]
			fmt.Println("Query:" + query)
			fmt.Println("Flag:" + flag)
			cmd := exec.Command("/usr/bin/grep", flag, query, fileName)
			resultsBytes,err := cmd.CombinedOutput()
			if err != nil {
				if err.Error() == "exit status 1" {
					resultsBytes = []byte("")
				} else {
					fmt.Println("Couldn't execute grep!")
				}
			}

			_,err = output.WriteString(string(resultsBytes))
			errorHandler(err, "Couldn't return results!", true)

			err = output.Flush()
			errorHandler(err, "Flush failed!", true)

		}(conn)
	}
}


