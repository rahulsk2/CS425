package main

import (
	"fmt"
	"os"
	"strings"
)


func main() {

	clientCheckArguments()
	port := os.Args[1]
	grepQuery := os.Args[2]
	var grepFlag string
	if len(os.Args) == 4 {
		grepFlag = os.Args[3]
	} else {
		grepFlag = "-E"
	}

	fmt.Println(port + " " + grepQuery + " " + grepFlag)


	messagechan := make(chan []string)
	hosts := []string{"wirelessprv-10-194-14-69.near.illinois.edu","wirelessprv-10-194-14-69.near.illinois.edu","wirelessprv-10-194-14-69.near.illinois.edu","wirelessprv-10-194-14-69.near.illinois.edu","wirelessprv-10-194-14-69.near.illinois.edu","wirelessprv-10-194-14-69.near.illinois.edu","wirelessprv-10-194-14-69.near.illinois.edu","wirelessprv-10-194-14-69.near.illinois.edu","wirelessprv-10-194-14-69.near.illinois.edu","wirelessprv-10-194-14-69.near.illinois.edu"}
	//targets := [3]string{"10.194.14.69:5000", "10.194.14.69:5001", "127.0.0.1:5002"}
	targets := [10]string{}
	for i, host := range hosts{
		resultBytes := dns_lookup(host)

		//TODO: Add the ports handling logic (either pass it in or just use hardcoded 5000)
		var strBuilder strings.Builder
		strBuilder.Write(resultBytes)
		strBuilder.WriteString(":500")
		strBuilder.WriteString(fmt.Sprintf("%d",i))
		targets[i] = strBuilder.String()
		fmt.Println("Added " + host + "(" + targets[i] + ") to list of machines to lookup")
	}


	MsgAllServersAlternate(targets,grepQuery,grepFlag,messagechan)


	for i := 0; i < len(targets); i++ {
		message := <-messagechan
		//TODO Add log file name logic (Currently hardcoded to 0)
		fmt.Println(fmt.Sprintf("\n\nTarget: %s(%s)\nLog File Name:machine.%d.log\nResponse:",reverse_dns_lookup(strings.Split(message[0], ":")[0]),message[0], 0))
		fmt.Println("---------------------------------------------------------------")
		for _, msg := range message[1:] {
			fmt.Println(msg)
		}
		fmt.Println(fmt.Sprintf("Lines Found: %d\n\n", len(message) - 1))
	}

}



func MsgAllServersAlternate(targets [10]string,text string, flags string, retchan chan []string) {
	for _, target := range targets {
		MsgServer(target,text,flags,retchan)
	}
}