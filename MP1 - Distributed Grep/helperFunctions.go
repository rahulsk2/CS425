package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"unicode"
)

func RemoveWhiteSpaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

func checkArguments(){
	if len(os.Args)!=3 {
		fmt.Println("Usage: "+os.Args[0]+" <index> <port>")
		os.Exit(0)
	}
}

func clientCheckArguments(){
	if len(os.Args) < 3 {
		fmt.Println("./MP1_Client_Application <port> <grepQuery> [grepFlag]")
		os.Exit(1)
	}
}

func errorHandler(errorObject error, errorMessage string, toExit bool){
	if errorObject != nil {
		fmt.Println(errorMessage)
		if toExit {
			os.Exit(1)
		}
	}
}

func dns_lookup(machine string) []byte{
	cmd := exec.Command("/usr/bin/dig","+short", machine)
	resultsBytes,err := cmd.CombinedOutput()
	resultsBytes = bytes.Trim(resultsBytes, "\n")
	errorHandler(err,"Couldn't lookup machine address:" + machine, true )
	return resultsBytes
}

func reverse_dns_lookup(ip string) []byte{
	cmd := exec.Command("/usr/bin/dig","+short", "-x", ip)
	resultsBytes,err := cmd.CombinedOutput()
	resultsBytes = bytes.Trim(resultsBytes, "\n")
	errorHandler(err,"Couldn't lookup ip address:" + ip, true )
	return resultsBytes
}


func MsgAllServers(targets [10]string,text string, flags string, retchan chan []string) {
	for _, target := range targets {
		MsgServer(target,text,flags,retchan)
	}
}

func MsgServer(target string, text string, flags string,retchan chan []string)  {
	go func() {
		results := make([]string,1)
		results[0] = target
		conn, err := net.Dial("tcp", target)
		errorHandler(err,"Couldn't connect to server: " + target , false )
		if err != nil{
			retchan <- append(results, "Couldn't connect to server")
			return
		}
		fmt.Fprintf(conn, text + ":" + flags + "\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			results = append(results, scanner.Text())
		}
		err =  scanner.Err()
		errorHandler(err,"Couldn't read server response!", true )
		retchan <- results
	}()
}