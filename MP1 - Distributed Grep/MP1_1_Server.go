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
	fname := "vm"+os.Args[1]+".log"
	port := os.Args[2]
	serv,err := net.Listen("tcp",":"+port)
	errorHandler(err,"Could not start server!", true)
	for {
		conn, err := serv.Accept()
		errorHandler(err, "Couldn't open connection", true)

		go func (conn net.Conn){
			defer conn.Close()
			input := bufio.NewReader(conn)
			output := bufio.NewWriter(conn)
			pattern,err := input.ReadString('\n')
			pattern=pattern[:len(pattern)-1]
			fmt.Println("Pattern:" + pattern)
			s := strings.Split(pattern, ":")
			text, flags := s[0], s[1]
			fmt.Println("Text:" + text)
			fmt.Println("Flags:" + flags)
			cmd := exec.Command("/usr/bin/grep",flags,text,fname)
			resultsBytes,err := cmd.CombinedOutput()
			if err != nil {
				if err.Error() == "exit status 1" {
					resultsBytes = []byte("")
				} else {
					fmt.Println("Couldn't execute grep!")
					//os.Exit(1)
				}
			}

			//var result bytes.Buffer
			//result.WriteString("Result:")
			//result.Write(resultsBytes)
			//fmt.Println(result.String())

			_,err = output.WriteString(string(resultsBytes))
			errorHandler(err, "Couldn't return results!", true)

			err = output.Flush()
			errorHandler(err, "Flush failed!", true)

		}(conn)
	}
}


