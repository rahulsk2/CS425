package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)




func main() {

	// Fetch the Port, Query and Flags from cmd line
	port := os.Args[1]

	//grepQueries := []string{"She sells sea shells on the sea shore", "The quick brown fox jumps over the lazy dog", "The five boxing wizards jump quickly", "hyzx", "hyzxC", "lFast"}
	grepQueries := []string{"x"}
	grepFlag := "E"

	messageChannel := make(chan []string) //Created a message channel for client server communication

	//Hardcoded list of VM hosts
	hosts := []string{"fa18-cs425-g38-01.cs.illinois.edu","fa18-cs425-g38-02.cs.illinois.edu","fa18-cs425-g38-03.cs.illinois.edu","fa18-cs425-g38-04.cs.illinois.edu","fa18-cs425-g38-05.cs.illinois.edu","fa18-cs425-g38-06.cs.illinois.edu","fa18-cs425-g38-07.cs.illinois.edu","fa18-cs425-g38-08.cs.illinois.edu","fa18-cs425-g38-09.cs.illinois.edu","fa18-cs425-g38-10.cs.illinois.edu"}

	//DNS lookup to resolve hostnames to ip addresses. Append port number at end.
	targets := [10]string{}
	for i, host := range hosts{
		resultBytes := dns_lookup(host)

		targets[i] =  string(resultBytes) + ":" + port
		fmt.Println("Added " + host + "(" + targets[i] + ") to list of machines to lookup.") //TODO: Possible cleanup
	}

	//Query the grep across all servers (happens in parallel)
	for index, grepQuery := range grepQueries {
		flag := false
		MsgAllServers(targets, grepQuery, grepFlag, messageChannel)

		for i := 0; i < len(targets); i++ {
			message := <-messageChannel
			fileName := "server_" + string(reverse_dns_lookup(strings.Split(message[0], ":")[0])) + port + strconv.Itoa(index+1)


			result := verify(message[1:], fileName)

			if !result{
				flag = true
				fmt.Println("Testcase " + strconv.Itoa(index) + " Failed. {" + fileName + "}")
			}
			if !flag{
				fmt.Println("Testcase " + strconv.Itoa(index) + " Passed.")
			}
		}

	}


}

func verify(response []string, fileName string) bool{
	file,err := os.Open(fileName)
	errorHandler(err,"Could not open filename!" + fileName, true)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		if count == len(response) || scanner.Text() != response[count]  {
			fmt.Println(scanner.Text())
			fmt.Println(response[count] )
			return false
		}
		count++
	}
	return true
}