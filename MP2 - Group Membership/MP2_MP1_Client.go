package main

import (
	"fmt"
	"os"
	"strings"
)




func main() {

	// Fetch the Port, Query and Flags from cmd line
	clientCheckArguments()
	port := os.Args[1]
	grepQuery := os.Args[2]
	var grepFlag string
	if len(os.Args) == 4 {
		grepFlag = os.Args[3]
	} else {
		grepFlag = "-E"
	}

	messageChannel := make(chan []string) //Created a message channel for client server communication

	//Hardcoded list of VM hosts
	hosts := []string{"fa18-cs425-g38-01.cs.illinois.edu","fa18-cs425-g38-02.cs.illinois.edu","fa18-cs425-g38-03.cs.illinois.edu","fa18-cs425-g38-04.cs.illinois.edu","fa18-cs425-g38-05.cs.illinois.edu","fa18-cs425-g38-06.cs.illinois.edu","fa18-cs425-g38-07.cs.illinois.edu","fa18-cs425-g38-08.cs.illinois.edu","fa18-cs425-g38-09.cs.illinois.edu","fa18-cs425-g38-10.cs.illinois.edu"}
	//hosts := []string{"10.194.14.69"}

	//DNS lookup to resolve hostnames to ip addresses. Append port number at end.
	targets := make([]string, len(hosts))
	for i, host := range hosts{
		resultBytes := dns_lookup(host)

		targets[i] = string(resultBytes) + ":" + port
		fmt.Println("Added " + host + "(" + targets[i] + ") to list of machines to lookup.") //TODO: Possible cleanup
	}
	//targets := []string{"10.194.14.69:6000"}

	//Query the grep across all servers (happens in parallel)
	MsgAllServers(targets,grepQuery,grepFlag, messageChannel)

	//Process each Query Result and display to user
	for i := 0; i < len(targets); i++ {
		message := <-messageChannel
		//TODO Add log file name logic
		fmt.Println(fmt.Sprintf("\n\nTarget: %s(%s)\nResponse:",reverse_dns_lookup(strings.Split(message[0], ":")[0]),message[0]))
		fmt.Println("---------------------------------------------------------------")
		for _, msg := range message[1:] {
			fmt.Println(msg)
		}
		fmt.Println(fmt.Sprintf("Lines Found: %d\n\n", len(message) - 1))
	}

}