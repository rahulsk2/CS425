package main

import (
	"bytes"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func sendMessageOverUDP(targetID string, message Message){
	//create the connection
	clientaddr, err := net.ResolveUDPAddr("udp", getIPfromID(targetID)+ ":"+ getPortfromID(targetID))
	ResolveError(err,false)
	conn, err := net.DialUDP("udp",nil,clientaddr)

	//Send the message
	jsonMessage := getJSONfromMessage(message)

	_, err = conn.Write(jsonMessage)
	ResolveError(err,false)
	conn.Close()
}


func ResolveError(e error, exitValue bool) {
	if e != nil {
		log.Println(e.Error())
		if exitValue{
			os.Exit(1)
		}
	}
}

func getIPfromID(memberID string) string{
	return strings.Split(memberID, "_")[1]
}

func getPortfromID(memberID string) string{
	return strings.Split(memberID, "_")[0]
}

func SDFSToReal(path string, version int) string{
	var buffer bytes.Buffer
	buffer.WriteString(sdfsDirName)
	buffer.WriteString("/")
	for _,charac := range path {
		switch charac {
		case 'c':
			buffer.WriteString("c1")
		case '/':
			buffer.WriteString("c2")
		default:
			buffer.WriteString(string(charac))
		}
	}
	buffer.WriteString("_version_")
	buffer.WriteString(strconv.Itoa(version))
	return buffer.String()
}

func RealToSDFS(path string) string{
	return path[len(sdfsDirName)+1:]
}

func DNS_lookup(machine string) []byte{
	cmd := exec.Command("/usr/bin/dig","+short", machine)
	resultsBytes,err := cmd.CombinedOutput()
	resultsBytes = bytes.Trim(resultsBytes, "\n")
	ResolveError(err, false )
	return resultsBytes
}

func shuffle(src []string) []string {
	final := make([]string, len(src))
	rand.Seed(time.Now().UTC().UnixNano())
	perm := rand.Perm(len(src))

	for i, v := range perm {
		final[v] = src[i]
	}
	return final
}

func buildSelfID() string {
	t := time.Now()
	return os.Args[2] + "_" + selfIPAddress + "_" + t.Format("20060102150405")
}

func GetLocalIP() string {
	interfaceAddresses, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range interfaceAddresses {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}