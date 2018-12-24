package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

/////////////////////////////////////
//	GLOBAL VARIABLES			   //
////////////////////////////////////
var selfIPAddress = GetLocalIP()
var selfID = buildSelfID()
var isIntroducer = false
const INTRODUCER_MACHINE = "fa18-cs425-g38-01.cs.illinois.edu"
var INTRODUCER_IP = string(DNS_lookup(INTRODUCER_MACHINE))
const INTRODUCER_PORT = 5000
var serverPortNumber int
//var logWritter *bufio.Writer

var input = make(chan Command)
var output = make(chan []string)
var killswitch = make(chan struct{})
var refocus = make(chan int)
/////////////////////////////////////
//	CUSTOM TYPES    			   //
////////////////////////////////////

type Command struct {
	cmd int //0 - Insert, 1 - Delete, 2 - ReadAll
	ID string
}

type Message struct {
	ID    string
	Type string
	Payload string
	AdditionalData []string
}


/////////////////////////////////////
//	MAIN FUNCTION				  //
////////////////////////////////////

func main() {
	mode, err := strconv.Atoi(os.Args[1])
	ResolveError(err,true)
	portnum,err := strconv.Atoi(os.Args[2])
	ResolveError(err,true)
	serverPortNumber = portnum

	//TODO: Remove Machine Log Port Number
	logfile,err := os.OpenFile("machine" + os.Args[2] + ".log",os.O_APPEND|os.O_CREATE,0644)
	ResolveError(err, true)
	//logWritter = bufio.NewWriter(logfile)

	mw := io.MultiWriter(os.Stdout, logfile)
	log.SetOutput(mw)

	//_, err = logWritter.WriteString("Writting to logfile\n")
	//ResolveError(err, true)
	log.Println("Starting logs for " + selfID)
	log.Println("Called with Mode:" + os.Args[1])


	go memberList()
	go clientManager()
	go PingServer(portnum)
	if mode==0 {
		go IntroduceServer(INTRODUCER_PORT)
	} else {

		selfaddress,err := net.ResolveUDPAddr("udp",":"+strconv.Itoa(portnum))
		ResolveError(err,false)
		//TODO: Add Introducer Port before ':' for demo
		introaddr,err := net.ResolveUDPAddr("udp",INTRODUCER_IP + ":" +strconv.Itoa(INTRODUCER_PORT))
		ResolveError(err,false)
		IntroduceClient(selfaddress,introaddr)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if scanner.Text()=="STOP" { //logic for leaving
			input<-Command{cmd:2,ID:""}
			mList:=<-output
			if len(mList) <=4 {
				for _,v := range mList {
					if v!=selfID {
						targetaddr,err := net.ResolveUDPAddr("udp",":"+getPortfromID(v))
						ResolveError(err,false)
						conn2,err := net.DialUDP("udp",nil,targetaddr)
						//send the FAIL message here
						stopMessage := createMessage(selfID, "LEAVE", selfID , []string{"None"})
						stopJSON := getJSONfromMessage(stopMessage)
						_, err = conn2.Write(stopJSON)
						ResolveError(err,false)
						conn2.Close()

					}
				}
			} else {
				selfidx:=0
				for i,v := range mList {
					if v==selfID {
						selfidx=i
						break
					}
				}
				targets:=make([]string,3)
				targets[0]=mList[(selfidx-1+len(mList))%len(mList)]
				targets[1]=mList[(selfidx+1+len(mList))%len(mList)]
				targets[2]=mList[(selfidx+1+len(mList))%len(mList)]
				for _,id := range targets {
					member_IP := getIPfromID(id)
					member_Port := getPortfromID(id)
					//send the FAIL message here
					targetaddr,err := net.ResolveUDPAddr("udp",member_IP + ":" + member_Port)
					ResolveError(err,false)
					conn2,err := net.DialUDP("udp",nil,targetaddr)
					//send the FAIL message here
					stopMessage := createMessage(selfID, "LEAVE", selfID , []string{"None"})
					stopJSON := getJSONfromMessage(stopMessage)
					_, err = conn2.Write(stopJSON)
					ResolveError(err,false)
					conn2.Close()
				}
			}
			return
		}
	}
}

/////////////////////////////////////
//	INTRODUCER 		 			  //
////////////////////////////////////
func IntroduceClient(laddr *net.UDPAddr, target *net.UDPAddr) {
	conn, err := net.DialUDP("udp",nil,target)
	ResolveError(err,true)
	defer conn.Close()
	//msg := make([]byte,1024)
	resp := make([]byte,1024)
	//prepare join message to be sent
	joinMessage := createMessage(selfID, "JOIN", "NONE", []string{"None"})
	_, err = conn.Write(getJSONfromMessage(joinMessage))
	ResolveError(err,false)
	n, err := conn.Read(resp)
	respJSON := []byte(string(resp[:n]))
	respMessage := getMessagefromJSON(respJSON)
	log.Println("Got Join Response From:" + respMessage.ID )
	log.Println(string(respJSON))
	//Populate Membership List from Response
	if respMessage.Type == "JOINACK" {
		membershipList := respMessage.AdditionalData
		for _,v := range membershipList {
			input <- Command{cmd:0,ID:v}
			<-output
		}
		for _, eachMember := range membershipList{
			member_IP := getIPfromID(eachMember)
			member_Port := getPortfromID(eachMember)
			if member_IP == selfIPAddress && member_Port == strconv.Itoa(serverPortNumber){
				continue
			}
		}
		refocus<-len(membershipList)
	}
}

func IntroduceServer(portnum int) {
	servaddr, err := net.ResolveUDPAddr("udp",":"+strconv.Itoa(portnum))
	ResolveError(err,false)
	serv, err := net.ListenUDP("udp",servaddr)
	ResolveError(err,true)
	defer serv.Close()
	buf := make([]byte,1024)
	log.Println("INTRODUCER READY!")
	input<-Command{cmd:0,ID:selfID}
	<-output

	//TODO: Update ListOfPossibleNodes With VM Machines IP:Port
	//Scan File of Nodes, Send INTRODUCE, get Introduce ACKS
	f, err := os.Open("ListOfPossibleNodes")
	ResolveError(err,true)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(),":")
		member_machine := strings.Trim(line[0], "\n")
		member_IP := string(DNS_lookup(member_machine))
		member_Port := strings.Trim(line[1], "\n")
		log.Println("Trying to see if " + member_machine + " [ " + member_IP + ":" + member_Port + " ] is alive")

		//INTRODUCE YOURSELF TO A NODE

		introduceMessage := createMessage(selfID, "INTRODUCE", selfID, []string{"None"})
		introduceMessageJSON := getJSONfromMessage(introduceMessage)
		if member_IP == selfIPAddress && member_Port == strconv.Itoa(serverPortNumber){
			continue
		}
		target, err := net.ResolveUDPAddr("udp",member_IP + ":" + member_Port)
		ResolveError(err,false)
		conn,err := net.DialUDP("udp",nil,target)
		ResolveError(err,false) //couldn't connect to the given target?
		_, err = conn.Write(introduceMessageJSON)
		ResolveError(err,false) //couldn't write to the given target?
		//WAIT FOR INTRODUCEACK FROM THE NODE
		conn.SetDeadline(time.Now().Add(2 * time.Second))
		resp := make([]byte,1024)
		n, err := conn.Read(resp)
		respJSON := []byte(string(resp[:n]))
		log.Println("Got:" + string(respJSON))
		if err!=nil{
			//INCASE OF TIMEOUT CONTINUE
			continue
		} else {
			//INCASE OF INTRODUCEACK, ADD TO MEMBERSHIP LIST
			responseMessage := getMessagefromJSON(respJSON)
			input <- Command{cmd: 0, ID: responseMessage.ID}
			<-output
		}
	}
	refocus<- 1
	for {
		n, addr, err := serv.ReadFromUDP(buf)

		//buf has the details of the node that joined
		ResolveError(err,false)
		joinRequestJSON := []byte(string(buf[:n]))
		joinRequestMessage := getMessagefromJSON(joinRequestJSON)


		if joinRequestMessage.Type == "INTRODUCEACK"{
			log.Println("Got an Introduce Ack:" + string(joinRequestJSON))
		} else {
			introduceMessage := createMessage(selfID, "INTRODUCE", joinRequestMessage.ID, []string{"None"})
			introduceMessageJSON := getJSONfromMessage(introduceMessage)
			input <- Command{cmd: 2, ID: ""}
			membershipList := <-output
			for _, member := range membershipList {
				member_IP := getIPfromID(member)
				member_Port := getPortfromID(member)
				//fmt.Println(member_Port + " - " + strconv.Itoa(serverPortNumber))
				if member_IP == selfIPAddress && member_Port == strconv.Itoa(serverPortNumber) {
					continue
				}
				clientaddr, err := net.ResolveUDPAddr("udp", member_IP+":"+member_Port)
				ResolveError(err,false)
				_, err = serv.WriteToUDP(introduceMessageJSON, clientaddr)
				ResolveError(err,false)
			}

			//update own membership list here
			input <- Command{cmd: 0, ID: joinRequestMessage.ID}
			<-output
			input <- Command{cmd: 2, ID: ""}
			membershipList = <-output
			//respond to join request
			joinResponse := createMessage(selfID, "JOINACK", "NONE", membershipList)
			_, err = serv.WriteToUDP(getJSONfromMessage(joinResponse), addr)
			ResolveError(err,false)

			//Start ping client to this node
			refocus <- 1
		}
	}
}

/////////////////////////////////////
//	PING CLIENT AND SERVER		//
////////////////////////////////////

func PingClient(target *net.UDPAddr,targetID string) {
	if targetID==selfID {
		panic("selfID bad!")
	}
	pingMessage := createMessage(selfID, "PING", "Nothing", []string{"None"})
	ping := getJSONfromMessage(pingMessage)
	resp := make([]byte,1024)
	ticker := time.NewTicker(2 * time.Second)
	stopflag := false
	for {
		select {
		case <-ticker.C:
			conn,err := net.DialUDP("udp",nil,target)
			ResolveError(err,false) //couldn't connect to the given target?
			_, err = conn.Write(ping)
			ResolveError(err,false) //couldn't write to the given target?
			conn.SetDeadline(time.Now().Add(2 * time.Second))
			_, err = conn.Read(resp)
			log.Println("Received: " + string(resp) + " From: " + targetID )
			if err!= nil {
				log.Println(targetID + " failed")
				ticker.Stop()
				stopflag = true
				input<-Command{cmd:1,ID:targetID}
				result:=<-output
				if result[0]=="TRUE" { //need to notify other servers
					input<-Command{cmd:2,ID:""}
					mList:=<-output
					selfidx := 0
					for i,v := range mList {
						if v==selfID {
							selfidx = i
							break
						}
					}
					if len(mList) <=3 {
						for _,v := range mList {
							if v!=selfID {
								clientaddr, err := net.ResolveUDPAddr("udp",getIPfromID(v) + ":"+ getPortfromID(v))
								ResolveError(err,false)
								conn2, err := net.DialUDP("udp",nil,clientaddr)
								ResolveError(err,false)
								//Send the failure messagee
								failMessage := createMessage(selfID, "FAILURE", targetID , []string{"None"})
								failJSON := getJSONfromMessage(failMessage)
								_, err = conn2.Write(failJSON)
								ResolveError(err,false)
								conn2.Close()
							}
						}
					} else { //need to send to 3 neighbours
						targets:=make([]string,3)
						targets[0]=mList[(selfidx-2+len(mList))%len(mList)]
						targets[1]=mList[(selfidx-1+len(mList))%len(mList)]
						targets[2]=mList[(selfidx+1+len(mList))%len(mList)]
						log.Println("New targets:"+targets[0]+","+targets[1]+","+targets[2])
						for _,v := range targets {
							clientaddr, err := net.ResolveUDPAddr("udp",getIPfromID(v) + ":"+ getPortfromID(v))
							ResolveError(err,false)
							conn3, err := net.DialUDP("udp",nil,clientaddr)
							ResolveError(err,false)
							//send a message indicating a failure here
							failMessage := createMessage(selfID, "FAILURE", targetID , []string{"None"})
							failJSON := getJSONfromMessage(failMessage)
							_, err = conn3.Write(failJSON)
							ResolveError(err,false)
							conn3.Close()
						}
					}

					refocus<- -1
				}


			}
			conn.Close()
		case <- killswitch:
			if !stopflag {
				ticker.Stop()
			}

			return

		}
	}
}

func PingServer(portnum int) {
	servaddr, err := net.ResolveUDPAddr("udp",":"+strconv.Itoa(portnum))
	ResolveError(err,false)
	serv, err := net.ListenUDP("udp",servaddr)
	ResolveError(err,true)
	defer serv.Close()
	buf := make([]byte,1024)
	fmt.Println("READY!")
	for {
		n,addr, err := serv.ReadFromUDP(buf)
		messageJSON := []byte(string(buf[:n]))
		message := getMessagefromJSON(messageJSON)
		log.Println("Received Message Type[" + message.Type + ", from ID:" + message.ID + ", Payload:" + message.Payload + "]")
		//buf now has the data that was sent
		ResolveError(err,false)
		//go Ack(addr)
		if message.Type == "PING" {
			ackMessage := createMessage(selfID, "ACK", "Nothing A", []string{"None"})
			_, err = serv.WriteToUDP(getJSONfromMessage(ackMessage), addr)
			ResolveError(err,false)
		} else if message.Type == "INTRODUCE" {
			log.Println("Received a Introduce to add " + message.Payload + " to membership list")
			input<-Command{cmd:0,ID:message.Payload}
			<-output
			refocus<-1
			ackMessage := createMessage(selfID, "INTRODUCEACK", "Nothing", []string{"None"})
			_, err = serv.WriteToUDP(getJSONfromMessage(ackMessage), addr)
			ResolveError(err,false)
		} else if message.Type == "FAILURE" || message.Type == "LEAVE" {
			fmt.Println("Received a " + message.Type + " from " + message.ID + " for " + message.Payload)
			input<-Command{cmd:1,ID:message.Payload}
			result:=<-output
			if result[0]=="TRUE" { //need to send failure info to neighbours
				input<-Command{cmd:2,ID:""}
				mList:=<-output
				selfidx := 0
				for i,v := range mList {
					if v==selfID {
						selfidx = i
						break
					}
				}
				if len(mList) <=3 {
					for _,v := range mList {
						if v!=selfID {
							clientaddr, err := net.ResolveUDPAddr("udp",getIPfromID(v) + ":"+ getPortfromID(v))
							ResolveError(err,false)
							conn2, err := net.DialUDP("udp",nil,clientaddr)
							//Send the failure messagee
							failMessage := createMessage(selfID, message.Type, message.Payload , []string{"None"})
							failJSON := getJSONfromMessage(failMessage)
							_, err = conn2.Write(failJSON)
							ResolveError(err,false)
							conn2.Close()
						}
					}
				} else { //need to send to 3 neighbours
					targets:=make([]string,3)
					targets[0]=mList[(selfidx-2+len(mList))%len(mList)]
					targets[1]=mList[(selfidx-1+len(mList))%len(mList)]
					targets[2]=mList[(selfidx+1+len(mList))%len(mList)]
					log.Println("New targets:"+targets[0]+","+targets[1]+","+targets[2])
					for _,v := range targets {
						clientaddr, err := net.ResolveUDPAddr("udp",getIPfromID(v) + ":"+ getPortfromID(v))
						ResolveError(err,false)
						conn3, err := net.DialUDP("udp",nil,clientaddr)
						//send a message indicating a failure here
						failMessage := createMessage(selfID, message.Type, message.Payload , []string{"None"})
						failJSON := getJSONfromMessage(failMessage)
						_, err = conn3.Write(failJSON)
						ResolveError(err,false)
						conn3.Close()
					}
				}
				refocus<- -1
			}
		}
	}
}


/////////////////////////////////////
//	UTILITY FUNCTIONS			  //
////////////////////////////////////
func clientManager() {
	pingclientcount := 0

	for {
		select {

		case <-refocus:
			for i := 0; i != pingclientcount; i++ {
				killswitch<-struct{}{}
			}
			//pingclientcount+=delta
			input<-Command{cmd:2,ID:""}
			mList:=<-output
			length := len(mList)
			if length <= 4 {
				pingclientcount = length-1
				for _,v := range mList {
					if v!= selfID {
						clientaddr, err := net.ResolveUDPAddr("udp",getIPfromID(v) + ":"+ getPortfromID(v))
						ResolveError(err,false)
						go PingClient(clientaddr,v)
					}
				}
			} else {
				pingclientcount = 3
				selfidx := 0
				for i,v := range mList {
					if v == selfID {
						selfidx=i
						break
					}
				}
				log.Println("Calculating new neighbours for " + selfID)
				targets:=make([]string,3)
				targets[0]=mList[(selfidx-2+len(mList))%len(mList)]
				targets[1]=mList[(selfidx-1+len(mList))%len(mList)]
				targets[2]=mList[(selfidx+1+len(mList))%len(mList)]
				log.Print("Neighbours: ")
				log.Println(targets)
				for _,v := range targets {
					clientaddr, err := net.ResolveUDPAddr("udp",getIPfromID(v) + ":"+ getPortfromID(v))
					ResolveError(err,false)
					go PingClient(clientaddr,v)
				}
			}
		}
	}
}
/*
This is used as an interface to interact with the membershiplist
 */
func memberList() {
	var membershipList []string
	for {
		select {
		case command:=<-input:
			if command.cmd==0 {
				membershipList=insertInSortedMembershipList(command.ID,membershipList)
				output<-[]string{}
			} else if command.cmd==1 { //delete node
				flag:=true
				for i, v := range membershipList {
					if v==command.ID {
						if i!=len(membershipList)-1 {
							membershipList=append(membershipList[:i],membershipList[i+1:]...)
						} else {
							membershipList=membershipList[:i]
						}
						output<-[]string{"TRUE"}
						flag=false
						break
					}
				}
				if flag {
					output<-[]string{"FALSE"}
				}
			} else {
				newSlice := make([]string,len(membershipList))
				copy(newSlice,membershipList)
				output<-newSlice
			}
		}
	}
}

func insertInSortedMembershipList(idToAdd string, existingList []string) []string{
	log.Print("Existing List was:")
	log.Println(existingList)
	log.Println("Trying to add:" + idToAdd)
	newList := append(existingList,idToAdd)
	sort.Strings(newList)
	log.Print("New List is:")
	log.Println(newList)
	return newList
}

func ResolveError(e error, exitValue bool) {
	if e != nil {
		log.Println(e.Error())
		if exitValue{
			os.Exit(1)
		}
	}
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

func createMessage(ID string, Type string, Payload string, AdditionalData []string) Message{
	message := Message{
		ID:    ID,
		Type:   Type,
		Payload: Payload,
		AdditionalData: AdditionalData,
	}
	return message
}

func getJSONfromMessage(message Message) []byte {
	var jsonMessage []byte
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
	}
	return jsonMessage
}

func getMessagefromJSON(jsonMessage []byte) Message {
	var message Message
	err := json.Unmarshal(jsonMessage, &message)
	if err != nil {
		log.Println(err)
	}
	//fmt.Println("")
	return message
}

func DNS_lookup(machine string) []byte{
	cmd := exec.Command("/usr/bin/dig","+short", machine)
	resultsBytes,err := cmd.CombinedOutput()
	resultsBytes = bytes.Trim(resultsBytes, "\n")
	ResolveError(err, false )
	return resultsBytes
}

func getIPfromID(memberID string) string{
	return strings.Split(memberID, "_")[1]
}

func getPortfromID(memberID string) string{
	return strings.Split(memberID, "_")[0]
}


func WriteToUDPWapper(c *net.UDPConn, b []byte, addr *net.UDPAddr) (int, error) {
	return c.WriteToUDP(b,addr)
}
func WriteWrapper(c *net.IPConn, b []byte) (int, error) {
	return c.Write(b)
}