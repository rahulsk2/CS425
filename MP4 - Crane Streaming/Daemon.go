package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
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


//Use this when running on VMs
const INTRODUCER_MACHINE = "fa18-cs425-g38-01.cs.illinois.edu"
var INTRODUCER_IP = string(DNS_lookup(INTRODUCER_MACHINE))

//USE this when running locally
//var INTRODUCER_IP = "127.0.0.1"


const INTRODUCER_PORT = 5000
var INTRODUCER_ID = ""

var startTime = time.Now()
var startTimeCrane = time.Now()

var serverPortNumber int
//var logWritter *bufio.Writer

var mListInput = make(chan Command)
var mListOutput = make(chan []string)
var killswitch = make(chan struct{})
var refocus = make(chan int)

var fschan = make(chan FsCommand,100)
var masterichan = make(chan MasterPacket,100)
var masterochan = make(chan FsRemoteMessage,100)
var masterfchan = make(chan string,100)
var masternchan = make(chan string,100)
var masterschan = make(chan string,100)
//var masterjchan = make(chan TopologyNodes,100)
var masterjchan = make(chan string,100)
var sdfsDirName = getPortfromID(selfID)
var fsmsgchan chan FsRemoteMessage = make(chan FsRemoteMessage)

var MASTER_ID string = ""

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

	mw := io.MultiWriter(os.Stdout, logfile)
	log.SetOutput(mw)


	go memberList()
	go clientManager()
	go PingServer(portnum)
	go FileSystem()
	if mode==0 {
		go IntroduceServer(INTRODUCER_PORT)
		INTRODUCER_ID = selfID
		MASTER_ID = ""
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
		if scanner.Text()=="stop" { //logic for leaving
			mList := MembershipListReadAll()
			if len(mList) <=4 {
				for _,v := range mList {
					if v!=selfID {
						//targetaddr,err := net.ResolveUDPAddr("udp",":"+getPortfromID(v))
						//ResolveError(err,false)
						//conn2,err := net.DialUDP("udp",nil,targetaddr)
						////send the FAIL message here
						stopMessage := createMessage(selfID, "LEAVE", selfID , []string{"None"})
						//stopJSON := getJSONfromMessage(stopMessage)
						//_, err = conn2.Write(stopJSON)
						//ResolveError(err,false)
						//conn2.Close()
						sendMessageOverUDP(v, stopMessage)

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
					//member_IP := getIPfromID(id)
					//member_Port := getPortfromID(id)
					////send the FAIL message here
					//targetaddr,err := net.ResolveUDPAddr("udp",member_IP + ":" + member_Port)
					//ResolveError(err,false)
					//conn2,err := net.DialUDP("udp",nil,targetaddr)
					////send the FAIL message here
					stopMessage := createMessage(selfID, "LEAVE", selfID , []string{"None"})
					//stopJSON := getJSONfromMessage(stopMessage)
					//_, err = conn2.Write(stopJSON)
					//ResolveError(err,false)
					//conn2.Close()
					sendMessageOverUDP(id, stopMessage)
				}
			}
			return
		} else {
			parts:=strings.Split(scanner.Text()," ")
			switch parts[0] {
			case "put":
				fschan <- FsCommand{Cmd: 0, SdfsFileName:parts[2], LocalFileName:parts[1], NumVersions:0}
			case "get":
				fschan <- FsCommand{Cmd: 1, SdfsFileName:parts[1], LocalFileName:parts[2], NumVersions:0}
			case "delete":
				fschan <- FsCommand{Cmd: 2, SdfsFileName:parts[1], LocalFileName:"", NumVersions:0}
			case "ls":
				fschan <- FsCommand{Cmd: 3, SdfsFileName:parts[1], LocalFileName:"", NumVersions:0}
			case "store":
				fschan <- FsCommand{Cmd: 4, SdfsFileName:"", LocalFileName:"", NumVersions:0}
			case "get-versions", "getversions", "get-version", "getversion":
				numversions, err := strconv.Atoi(parts[3])
				ResolveError(err, true)
				fschan <- FsCommand{Cmd: 5, SdfsFileName:parts[1], LocalFileName:parts[2], NumVersions:numversions}
			case "list":
				mListInput <-Command{cmd: 2,ID:""}
				mList:=<-mListOutput
				fmt.Println("Membership List:" , mList)
			case "selfid":
				fmt.Println("Self ID:", selfID)
			case "masterid":
				fmt.Println("Master ID:", MASTER_ID)
			case "introducerid":
				fmt.Println("Introducer ID:", INTRODUCER_ID)
			case "submit":
				subjob <- parts[1]
			}
		}
	}
}

var subjob = make(chan string)

var jobchan = make(chan TopologyNodeWithMachines,20)
var srcchan = make(chan ContactMessage,20)
func JobSystem() {
	InputWaitList := make([]ContactMessage,0)
	InputWaitSize := 0
	var tmBuffer TopologyNodeWithMachines
	for {
		select {
		case topofname := <-subjob:
			serv, err := net.Listen("tcp",":0")
			ResolveError(err,true)
			addr := serv.Addr()
			addr_string := addr.String()
			addr_split := strings.Split(addr_string, ":")
			port_num := addr_split[len(addr_split)-1]
			target_to_recv_from := getIPfromID(selfID) + ":"+ port_num
			msg := createMessage(selfID, "JOB_CREATE", target_to_recv_from, []string{""})
			go func() {
				nodesData := ReadTopologyFile(topofname)
				sock, err := serv.Accept()
				defer sock.Close()
				ResolveError(err,true)
				fmt.Println("NodesData:", string([]byte(nodesData)))
				sock.Write([]byte(nodesData))
			}()
			sendMessageOverUDP(MASTER_ID, msg)
		case tm := <-jobchan:
			tmBuffer = tm
			InputWaitSize = len(tm.InputBolts)
			if tm.OpType == "SINK" && len(InputWaitList) == InputWaitSize{ //input list has been filled
				ctx,cancel := context.WithCancel(context.Background())
				fmt.Println("Assigned Job:", tmBuffer)
				err:=os.Remove(tmBuffer.FileIO)
				ResolveError(err,false)
				fmt.Println("Removed outfile!")
				terminator := make(chan bool)
				completetype := true
				go func(count int) {
					for i := 0; i!=count; i++ {
						completetype = completetype && <-terminator
					}
					if completetype {
						fmt.Println("Sink Finished Processing")
						donemessage := createMessage(selfID,"JOB_DONE","",[]string{""})
						//tmBuffer.FileIO
						fschan <- FsCommand{Cmd: 0, SdfsFileName:tmBuffer.FileIO, LocalFileName:tmBuffer.FileIO, NumVersions:0}
						fmt.Println("Saved Output File in SDFS..", tmBuffer.FileIO)
						sendMessageOverUDP(MASTER_ID,donemessage)
						InputWaitSize = 0
						InputWaitList = make([]ContactMessage,0)
					} else {
						fmt.Println("Detecting error.. Shutting down..")
						InputWaitSize = 0
						InputWaitList = make([]ContactMessage,0)
					}
				} (InputWaitSize)
				for _,v := range InputWaitList {
					conn,err := net.Dial("tcp",v.Addr)
					//ResolveError(err,true)
					if err != nil { //a fast failure has occurred
						cancel()
						fmt.Println("I am cancelling")
						terminator<-false
						continue
					}
					go func(conn net.Conn){
						defer conn.Close()
						linereader := bufio.NewReader(conn)
						outfile,err := os.OpenFile(tmBuffer.FileIO, os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0777)
						defer outfile.Close()
						ResolveError(err,true)
						for {
							select {
							case <-ctx.Done():
								terminator<-false
								return
							default:
								line,err := linereader.ReadString('\n')
								if err != nil {
									cancel()
									fmt.Println("I am cancelling")
									terminator<-false
									return
								}
								if line == "TOFFEE\n" {
									terminator<-true
									return
								} else {
									_,err := outfile.Write([]byte(line))
									ResolveError(err,true)
								}
							}
						}
					}(conn)
				}
			} else if len(InputWaitList) == InputWaitSize {
				//we want to contact our output bolts
				//input list has been filled, so begin duties
				ctx, cancel := context.WithCancel(context.Background())
				completetype := true
				terminator := make(chan bool)
				fmt.Println("Assigned Job:", tmBuffer)
				//wait at this point for all the output connections to be set up
				serv, err := net.Listen("tcp",":0")
				ResolveError(err,true)
				addr := serv.Addr()
				addr_string := addr.String()
				addr_split := strings.Split(addr_string, ":")
				port_num := addr_split[len(addr_split)-1]
				target_to_recv_from := getIPfromID(selfID) + ":"+ port_num
				//convert address into actual address
				msg := createMessage(selfID,"JOB_CONTACT",target_to_recv_from,[]string{""})
				for _, eachOutputBolt := range tmBuffer.OutputBolts {
					//fmt.Println(eachOutputBolt)
					sendMessageOverUDP(eachOutputBolt, msg)
				}
				//send said address to each of the output bolts here
				//we assume that the job file is present on all nodes
				targetchans := make([]chan string,len(tmBuffer.OutputBolts))
				for i,_ := range targetchans {
					targetchans[i] = make(chan string)
				}
				//havesetupalloutputchans := make(chan struct{})
				//go func(serv net.Listener,childcount int) {
				childcount := len(tmBuffer.OutputBolts)
				for i := 0; i < childcount; i++ {
					conn,err := serv.Accept()
					ResolveError(err,true)
					go func(conn net.Conn, target chan string) {
						//defer conn.Close()
						goingOn := true
						for {
							//connWriter := bufio.NewWriter(conn)
							select {
							case <-ctx.Done():
								goingOn = false
								conn.Close()
							case v,ok := <-target:
								if !ok && goingOn { //job finished!
									terminator<- true
									return
								} else if ok && goingOn {
									//conn.SetWriteDeadline(time.Now().Add(2*time.Second))
									_,err := conn.Write([]byte(v))
									//_,err:=connWriter.Write([]byte(v))
									//_,err := connWriter.WriteString(v)
									//err1:=connWriter.Flush()
									//conn.SetWriteDeadline(time.Time{})
									//fmt.Println(n)
									if err!=nil /*|| err1 != nil*/ {
										cancel()
										fmt.Println("I am cancelling")
										goingOn = false
										conn.Close()
										//ResolveError(err,false)
									}
								} else if ok && !goingOn { //drop the line quietly
									//
								} else if !ok && !goingOn {
									terminator <- false
									return
								}
							}
						}
					}(conn,targetchans[i])
				}
				//havesetupalloutputchans<-struct{}{}
				//}(serv,len(tmBuffer.OutputBolts))
				//we want to wait to be contacted by our input bolts
				InputWaitSize = len(tmBuffer.InputBolts)
				//<-havesetupalloutputchans
				BoltCmd := exec.Command("python3",tmBuffer.FileToExecute,strconv.Itoa(tmBuffer.MethodToExecute),tmBuffer.OpType,tmBuffer.FileIO,"0")
				BoltOutputPipe,err := BoltCmd.StdoutPipe()
				ResolveError(err,true)
				BoltInputPipe,err := BoltCmd.StdinPipe()
				ResolveError(err,true)
				BoltReader := bufio.NewReader(BoltOutputPipe)
				BoltWriter := bufio.NewWriter(BoltInputPipe)
				BoltWriteChan := make(chan string)
				BoltCloseChan := make(chan struct{})
				go func() {
					counter := InputWaitSize
					for {
						select {
						case <-ctx.Done():
							BoltWriter.WriteString("TOFFEE\n")
							BoltWriter.Flush()
							return
						case str := <-BoltWriteChan:
							BoltWriter.WriteString(str)
							BoltWriter.Flush()
						case <-BoltCloseChan:
							counter--
							//fmt.Println("Closed a port!",counter)
							if counter==0 {
								//fmt.Println("SVTFOE")
								//all input sources are done
								BoltWriter.WriteString("TOFFEE\n")
								BoltWriter.Flush()
								return
							}
						}
					}
				} ()
				BoltCmd.Start()
				go func() {
					for {
						select {
						case <-ctx.Done():
							BoltCmd.Process.Kill()
							for i := 0; i != len(tmBuffer.OutputBolts); i++ {
								close(targetchans[i])
							}
							return
						default:
							line,err := BoltReader.ReadString('\n')
							ResolveError(err,true)
							for i := 0; i != len(tmBuffer.OutputBolts); i++ {
								targetchans[i] <- line
							}
							if line=="TOFFEE\n" {
								BoltCmd.Wait()
								for i := 0; i != len(tmBuffer.OutputBolts); i++ {
									close(targetchans[i])
								}
								return
							} else {
								//fmt.Println("line:", line)
							}
						}
					}
				} ()
				for _,v := range InputWaitList {
					conn,err := net.Dial("tcp",v.Addr)
					//ResolveError(err,true)
					if err != nil { //a fast failure has occurred
						cancel()
						fmt.Println("I am cancelling")
						terminator<-false
						//BoltCloseChan <- struct{}{}
						continue
					}
					go func(conn net.Conn){
						linereader := bufio.NewReader(conn)
						//outfile,err := os.OpenFile(tmBuffer.FileIO, os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0777)
						ResolveError(err,true)
						//defer outfile.Close()
						for {
							select {
							case <-ctx.Done():
								//BoltCloseChan <- struct{}{}
								terminator<-false
								return
							default:
								line,err := linereader.ReadString('\n')
								if err != nil {
									cancel()
									fmt.Println("I am cancelling")
									//BoltCloseChan <- struct{}{}
									terminator<-false
									//ResolveError(err,false)
									return
								}
								if line == "TOFFEE\n" {
									BoltCloseChan <- struct{}{}
									terminator<-true
									return
								} else {
									BoltWriteChan <- line
								}
							}
						}
					}(conn)
				}
				for idx := 0; idx != len(tmBuffer.OutputBolts) + len(tmBuffer.InputBolts); idx++ {
					completetype = completetype && <-terminator
				}
				if completetype {
					fmt.Println("Finished Processing..")
					donemessage := createMessage(selfID,"JOB_DONE","",[]string{""})
					sendMessageOverUDP(MASTER_ID,donemessage)
					InputWaitSize = 0
					InputWaitList = make([]ContactMessage,0)
				} else {
					fmt.Println("Detected Error.. shutting down job..")
					InputWaitSize = 0
					InputWaitList = make([]ContactMessage,0)
				}
			}
		case sm := <-srcchan:
			InputWaitList = append(InputWaitList,sm)
			if len(InputWaitList) == InputWaitSize && tmBuffer.OpType == "SINK" { //input list has been filled
				ctx,cancel := context.WithCancel(context.Background())
				fmt.Println("Assigned Job:", tmBuffer)
				err:=os.Remove(tmBuffer.FileIO)
				ResolveError(err,false)
				fmt.Println("Removed outfile!")
				terminator := make(chan bool)
				completetype := true
				go func(count int) {
					for i := 0; i!=count; i++ {
						completetype = completetype && <-terminator
					}
					if completetype {
						fmt.Println("Sink Finished Processing..")
						donemessage := createMessage(selfID,"JOB_DONE","",[]string{""})
						sendMessageOverUDP(MASTER_ID,donemessage)
						fschan <- FsCommand{Cmd: 0, SdfsFileName:tmBuffer.FileIO, LocalFileName:tmBuffer.FileIO, NumVersions:0}
						fmt.Println("Saved Output File in SDFS..", tmBuffer.FileIO)
						InputWaitSize = 0
						InputWaitList = make([]ContactMessage,0)
					} else {
						fmt.Println("Detected error.. Shutting Down..")
						InputWaitSize = 0
						InputWaitList = make([]ContactMessage,0)
					}
				} (InputWaitSize)
				for _,v := range InputWaitList {
					conn,err := net.Dial("tcp",v.Addr)
					//ResolveError(err,true)
					if err != nil { //a fast failure has occurred
						cancel()
						fmt.Println("I am cancelling")
						terminator<-false
						continue
					}
					go func(conn net.Conn){
						defer conn.Close()
						linereader := bufio.NewReader(conn)
						outfile,err := os.OpenFile(tmBuffer.FileIO, os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0777)
						defer outfile.Close()
						ResolveError(err,true)
						for {
							select {
							case <-ctx.Done():
								terminator<-false
								return
							default:
								line,err := linereader.ReadString('\n')
								if err != nil {
									cancel()
									fmt.Println("I am cancelling")
									terminator<-false
									return
								}
								if line == "TOFFEE\n" {
									terminator<-true
									return
								} else {
									_,err := outfile.Write([]byte(line))
									ResolveError(err,true)
								}
							}
						}
					}(conn)
				}
			} else if len(InputWaitList) == InputWaitSize { //input list has been filled, so begin duties
				ctx, cancel := context.WithCancel(context.Background())
				completetype := true
				terminator := make(chan bool)
				fmt.Println("Starting a job..", tmBuffer)
				//wait at this point for all the output connections to be set up
				serv, err := net.Listen("tcp",":0")
				ResolveError(err,true)
				addr := serv.Addr()
				addr_string := addr.String()
				addr_split := strings.Split(addr_string, ":")
				port_num := addr_split[len(addr_split)-1]
				target_to_recv_from := getIPfromID(selfID) + ":"+ port_num
				//convert address into actual address
				msg := createMessage(selfID,"JOB_CONTACT",target_to_recv_from,[]string{""})
				for _, eachOutputBolt := range tmBuffer.OutputBolts {
					//fmt.Println(eachOutputBolt)
					sendMessageOverUDP(eachOutputBolt, msg)
				}
				//send said address to each of the output bolts here
				//we assume that the job file is present on all nodes
				targetchans := make([]chan string,len(tmBuffer.OutputBolts))
				for i,_ := range targetchans {
					targetchans[i] = make(chan string)
				}
				//havesetupalloutputchans := make(chan struct{})
				//go func(serv net.Listener,childcount int) {
				childcount := len(tmBuffer.OutputBolts)
				for i := 0; i < childcount; i++ {
					conn,err := serv.Accept()
					ResolveError(err,true)
					go func(conn net.Conn, target chan string) {
						//defer conn.Close()
						//connWriter := bufio.NewWriter(conn)
						goingOn := true
						for {
							select {
							case <-ctx.Done():
								goingOn = false
								conn.Close()
							case v,ok := <-target:
								if !ok && goingOn { //job finished!
									terminator<- true
									return
								} else if ok && goingOn {
									//conn.SetWriteDeadline(time.Now().Add(2*time.Second))
									_,err := conn.Write([]byte(v))
									//_,err := connWriter.WriteString(v)
									//err1:=connWriter.Flush()
									//conn.SetWriteDeadline(time.Time{})
									//fmt.Println(n)
									if err!=nil /*|| err1 != nil*/ {
										cancel()
										fmt.Println("I am cancelling")
										goingOn = false
										conn.Close()
										//ResolveError(err,false)
									}
								} else if ok && !goingOn { //drop the line quietly
									//
								} else if !ok && !goingOn {
									terminator <- false
									return
								}
							}
						}
					}(conn,targetchans[i])
				}
				//havesetupalloutputchans<-struct{}{}
				//}(serv,len(tmBuffer.OutputBolts))
				//we want to wait to be contacted by our input bolts
				InputWaitSize = len(tmBuffer.InputBolts)
				//<-havesetupalloutputchans
				BoltCmd := exec.Command("python3",tmBuffer.FileToExecute,strconv.Itoa(tmBuffer.MethodToExecute),tmBuffer.OpType,tmBuffer.FileIO,"0")
				BoltOutputPipe,err := BoltCmd.StdoutPipe()
				ResolveError(err,true)
				BoltInputPipe,err := BoltCmd.StdinPipe()
				ResolveError(err,true)
				BoltReader := bufio.NewReader(BoltOutputPipe)
				BoltWriter := bufio.NewWriter(BoltInputPipe)
				BoltWriteChan := make(chan string)
				BoltCloseChan := make(chan struct{})
				go func() {
					counter := InputWaitSize
					for {
						select {
						case <-ctx.Done():
							BoltWriter.WriteString("TOFFEE\n")
							BoltWriter.Flush()
							return
						case str := <-BoltWriteChan:
							BoltWriter.WriteString(str)
							BoltWriter.Flush()
						case <-BoltCloseChan:
							counter--
							//fmt.Println("Closed a port!",counter)
							if counter==0 {
								//all input sources are done
								//fmt.Println("SVTFOE")
								BoltWriter.WriteString("TOFFEE\n")
								BoltWriter.Flush()
								return
							}
						}
					}
				} ()
				BoltCmd.Start()
				go func() {
					for {
						select {
						case <-ctx.Done():
							BoltCmd.Process.Kill()
							for i := 0; i != len(tmBuffer.OutputBolts); i++ {
								close(targetchans[i])
							}
							return
						default:
							line,err := BoltReader.ReadString('\n')
							ResolveError(err,true)
							for i := 0; i != len(tmBuffer.OutputBolts); i++ {
								targetchans[i] <- line
							}
							if line=="TOFFEE\n" {
								BoltCmd.Wait()
								for i := 0; i != len(tmBuffer.OutputBolts); i++ {
									close(targetchans[i])
								}
								return
							}
						}
					}
				} ()
				for _,v := range InputWaitList {
					conn,err := net.Dial("tcp",v.Addr)
					//ResolveError(err,true)
					if err != nil { //a fast failure has occurred
						cancel()
						fmt.Println("I am cancelling")
						terminator<-false
						//BoltCloseChan <- struct{}{}
						continue
					}
					go func(conn net.Conn){
						linereader := bufio.NewReader(conn)
						//outfile,err := os.OpenFile(tmBuffer.FileIO, os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0777)
						ResolveError(err,true)
						//defer outfile.Close()
						for {
							select {
							case <-ctx.Done():
								terminator<-false
								//BoltCloseChan <- struct{}{}
								return
							default:
								line,err := linereader.ReadString('\n')
								if err != nil {
									cancel()
									fmt.Println("I am cancelling",line)
									terminator<-false
									//BoltCloseChan <- struct{}{}
									//ResolveError(err,false)
									return
								}
								if line == "TOFFEE\n" {
									BoltCloseChan <- struct{}{}
									terminator<-true
									return
								} else {
									BoltWriteChan <- line
								}
							}
						}
					}(conn)
				}
				for idx := 0; idx != len(tmBuffer.OutputBolts) + len(tmBuffer.InputBolts); idx++ {
					completetype = completetype && <-terminator
				}
				if completetype {
					fmt.Println("Done Processing the job..")
					donemessage := createMessage(selfID,"JOB_DONE","",[]string{""})
					sendMessageOverUDP(MASTER_ID,donemessage)
					InputWaitSize = 0
					InputWaitList = make([]ContactMessage,0)
				} else {
					fmt.Println("Detected error.. Shutting down..")
					InputWaitSize = 0
					InputWaitList = make([]ContactMessage,0)
				}
			}
		}
	}
}

func MasterServer() {
	filedata := make(map[string]FileMetaData)
	nodedata := make(map[string]map[string]bool)
	boltallot := make(map[int]string) //maps bolts to nodes
	revboltallot := make(map[string][]int) //reverse map of above
	nodedata[selfID]=make(map[string]bool)
	var unintroduced =make(map[string]bool)
	hasAllIntroduced := true
	jobRefCount := -1
	var job TopologyNodes
	//get replies
	//build file distribution table
	for {
		select {
		case jobBackup:=<-masterschan:
			fmt.Println("I'm a backup!!!",jobBackup)
			//if message tells it to drop data, clear boltallot, revboltallot and job, jobRefCount
			//else, set those fields from data
		case jID:=<-masterjobdonechan:
			jobRefCount--
			fmt.Println("Job finished at:",jID)
			if jobRefCount==0 {
				fmt.Println("Overall Job Completed!")
				time_elapsed_crane := time.Since(startTimeCrane)
				fmt.Println("Time Taken For Job:", time_elapsed_crane)
				boltallot=make(map[int]string)
				revboltallot=make(map[string][]int)
				jobRefCount=-1
				//master tells standby to drop data
				estimMinID := "99999999999"
				mList := MembershipListReadAll()
				for _, v := range mList {
					if v != INTRODUCER_ID && v != selfID {
						if v < estimMinID {
							estimMinID = v
						}
					}
				}
				//boltallot,revboltallot,job,jobRefCount
				payload := "ALL CLEAR"
				standbyMsg := createMessage(selfID,"JOB_STANDBY",payload,[]string{""})
				sendMessageOverUDP(estimMinID,standbyMsg)
				//send job spec to the hot standby
			} else {
				estimMinID := "99999999999"
				mList := MembershipListReadAll()
				for _, v := range mList {
					if v != INTRODUCER_ID && v != selfID {
						if v < estimMinID {
							estimMinID = v
						}
					}
				}
				//boltallot,revboltallot,job,jobRefCount
				payload := ""
				standbyMsg := createMessage(selfID,"JOB_STANDBY",payload,[]string{""})
				sendMessageOverUDP(estimMinID,standbyMsg)
			}
			//send job finished notif to hot standby
		case jobContact:=<-masterjchan:
			conn, err := net.Dial("tcp",jobContact)
			ResolveError(err,true)
			var buf bytes.Buffer
			io.Copy(&buf,conn)
			conn.Close()
			readData:=buf.Bytes()
			job= ConvertNodesDataToNodes(string(readData))
			jobRefCount = len(job.TopologyNodes)
			mList := shuffle(MembershipListReadAll())
			mListIter := 0
			for _, tn := range job.TopologyNodes {
				for mList[mListIter] == INTRODUCER_ID || mList[mListIter] == MASTER_ID {
					mListIter++
					if mListIter==len(mList) {
						mListIter=0
					}
				}
				nodeID := mList[mListIter]
				boltallot[tn.ID]=nodeID
				if _,ok := revboltallot[nodeID]; ok {
					revboltallot[nodeID]=append(revboltallot[nodeID],tn.ID)
				} else {
					revboltallot[nodeID]=[]int{tn.ID}
				}

				mListIter++
			}
			startTimeCrane = time.Now()
			fmt.Println("Assigned the job to nodes: ", boltallot)
			for _, tn := range job.TopologyNodes {
				//build the message to send that node here
				//allot job to a node
				tnm := convertTopologyNodeToTopologyNodesWithMachines(tn, boltallot)
				payload := string(getJSONfromTopologyNodeWithMachines(tnm))
				message := createMessage(selfID,"JOB_ALLOT",payload,[]string{""})
				sendMessageOverUDP(boltallot[tn.ID],message)
			}
			estimMinID := "99999999999"
			for _, v := range mList {
				if v != INTRODUCER_ID && v != selfID {
					if v < estimMinID {
						estimMinID = v
					}
				}
			}
			//boltallot,revboltallot,job,jobRefCount
			payload := ""
			standbyMsg := createMessage(selfID,"JOB_STANDBY",payload,[]string{""})
			sendMessageOverUDP(estimMinID,standbyMsg)
			//send job spec to the hot standby
		case packet:=<-masterichan:
			switch packet.Cmd {
			case -1:
				jsonSdfsDir := packet.SdfsFileName
				sdfsFileNamesDir := getMapfromJSON([]byte(jsonSdfsDir))
				nodedata[packet.ID] = make(map[string]bool)
				for fname, lv := range sdfsFileNamesDir {
					fmd, ok := filedata[fname]
					if ok {
						fmd.ReplicaList=append(fmd.ReplicaList,packet.ID)
						if fmd.LatestVersion<lv {
							fmd.LatestVersion=lv
						}
					} else {
						fmd=FileMetaData{LatestVersion:lv,ReplicaList:[]string{packet.ID}}
					}
					filedata[fname]=fmd
					nodedata[packet.ID][fname]=true
				}
				delete(unintroduced,packet.ID)
				if len(unintroduced) == 0 && !hasAllIntroduced {
					hasAllIntroduced = true
					mList := MembershipListReadAll()
					var cutoff int
					if len(mList)>=5 {
						cutoff=4
					} else {
						cutoff=len(mList)-1
					}
					for file,metadata := range filedata {
						if len(metadata.ReplicaList)<cutoff {
							replicaSet := make(map[string]bool)
							for _,v := range metadata.ReplicaList {
								replicaSet[v]=true
							}
							newReplicas:=make([]string,0)
							for _,v := range mList {
								if _,ok := replicaSet[v];!ok && v != INTRODUCER_ID {
									newReplicas=append(newReplicas,v)
									if (len(newReplicas)+len(metadata.ReplicaList))>=cutoff {
										break
									}
								}
							}
							payload := getJSONfromFsRemoteMessage(FsRemoteMessage{Cmd:0,SdfsFileName:file,Data:metadata.ReplicaList,LatestVersion:metadata.LatestVersion})
							message := createMessage(selfID,"FILE_GENERAL",string(payload),[]string{""})
							//msgJSON := getJSONfromMessage(message)
							for _,v := range newReplicas {
								//targetaddr,err := net.ResolveUDPAddr("udp",getIPfromID(v)+":"+getPortfromID(v))
								//ResolveError(err,true)
								//conn,err := net.DialUDP("udp",nil,targetaddr)
								//ResolveError(err,true)
								//_,err = conn.Write(msgJSON)
								//ResolveError(err,true)
								//conn.Close()
								sendMessageOverUDP(v, message)
								nodedata[v][file]=true
							}
							metadata.ReplicaList=append(metadata.ReplicaList,newReplicas...)
							filedata[file]=metadata
						}
					}
					//look at every file
					//if number of replicas of the file is < min(4,number of file storing nodes)
					//rebalance for that file
				}
			case 0:
				//PUT
				if fmd,ok := filedata[packet.SdfsFileName]; ok { //update to existing file
					//return fmd.replicas, except sender, and the updated version number for this
					modded := filedata[packet.SdfsFileName]
					modded.LatestVersion++
					filedata[packet.SdfsFileName]=modded
					masterochan<-FsRemoteMessage{Cmd: -1, SdfsFileName:packet.SdfsFileName, Data: fmd.ReplicaList, LatestVersion:filedata[packet.SdfsFileName].LatestVersion}
				} else { //brand new file
					mList := MembershipListReadAll()
					replicas := make([]string, 0)
					rCount := 0
					for _,v := range shuffle(mList) {
						if v != INTRODUCER_ID {
							replicas=append(replicas,v)
							nodedata[v][packet.SdfsFileName]=true
							rCount++
						}
						if rCount == 4 {
							break
						}
					}
					filedata[packet.SdfsFileName]=FileMetaData{LatestVersion: 1, ReplicaList:replicas}
					response := FsRemoteMessage{Cmd: -1, SdfsFileName:packet.SdfsFileName, Data: replicas, LatestVersion:filedata[packet.SdfsFileName].LatestVersion}
					masterochan<-response//generate 3 random targets and return them, after updating fmd.replicas, and version number
				}
			case 1:
				if fmd,ok := filedata[packet.SdfsFileName]; ok { //valid file
					response := FsRemoteMessage{Cmd: -2, SdfsFileName:packet.SdfsFileName, Data: fmd.ReplicaList[:1], LatestVersion:filedata[packet.SdfsFileName].LatestVersion}
					masterochan<-response
					//return fmd.replicas
				} else { //erroneous request
					masterochan<-FsRemoteMessage{Cmd: -400, SdfsFileName:packet.SdfsFileName, Data: []string{""}, LatestVersion:0}
				}
			case 2:
				if fmd,ok := filedata[packet.SdfsFileName]; ok { //file exists to be deleted
					response := FsRemoteMessage{Cmd:-3,SdfsFileName:packet.SdfsFileName,Data: fmd.ReplicaList, LatestVersion:fmd.LatestVersion}
					masterochan<-response
					for _,v := range filedata[packet.SdfsFileName].ReplicaList {
						delete(nodedata[v],packet.SdfsFileName)
					}
					delete(filedata,packet.SdfsFileName)
					//send delete message to all the replicas except sender
				} else { //erroneous request
					masterochan<-FsRemoteMessage{Cmd: -400, SdfsFileName:packet.SdfsFileName, Data: []string{""}, LatestVersion:0}
				}
			case 3:
				if fmd,ok := filedata[packet.SdfsFileName]; ok { //ls sdfsfilename
					modded := filedata[packet.SdfsFileName]
					filedata[packet.SdfsFileName]=modded
					masterochan<-FsRemoteMessage{Cmd: -4, SdfsFileName:packet.SdfsFileName, Data: fmd.ReplicaList, LatestVersion:filedata[packet.SdfsFileName].LatestVersion}
				} else {
					masterochan<-FsRemoteMessage{Cmd: -400, SdfsFileName:packet.SdfsFileName, Data: []string{""}, LatestVersion:0}
				}
				//case 4:
				//	//for all files that were originally associated with that node, assign 1 new replica node
			case 5:
				//	//return list of replicas and what the latest version is
				if fmd,ok := filedata[packet.SdfsFileName]; ok { //valid file
					response := FsRemoteMessage{Cmd: -5, SdfsFileName:packet.SdfsFileName, Data: fmd.ReplicaList[:1], LatestVersion:filedata[packet.SdfsFileName].LatestVersion}
					masterochan<-response
					//return fmd.replicas
				} else { //erroneous request
					masterochan<-FsRemoteMessage{Cmd: -400, SdfsFileName:packet.SdfsFileName, Data: []string{""}, LatestVersion:0}
				}
			}
		case failedID:= <-masterfchan:
			if (MASTER_ID==selfID) {
				for fname,_ := range nodedata[failedID] {
					replicaSet := make(map[string]bool)
					realReplicas := make([]string,0)
					for _,elem := range filedata[fname].ReplicaList {
						if elem!=failedID {
							replicaSet[elem]=true
							realReplicas=append(realReplicas,elem)
						}
					}
					mList := MembershipListReadAll()
					for _, v := range shuffle(mList) {
						if _,ok := replicaSet[v]; !ok && v!=INTRODUCER_ID && v!= failedID {
							payload:=FsRemoteMessage{Cmd:0,SdfsFileName:fname,Data:realReplicas,LatestVersion:filedata[fname].LatestVersion}
							msg := createMessage(selfID,"FILE_GENERAL",string(getJSONfromFsRemoteMessage(payload)),[]string{""})
							{
								//targetaddr,err := net.ResolveUDPAddr("udp",getIPfromID(v) + ":"+ getPortfromID(v))
								//ResolveError(err,false)
								//conn2,err := net.DialUDP("udp",nil,targetaddr)
								//msgJSON := getJSONfromMessage(msg)
								//_, err = conn2.Write(msgJSON)
								//ResolveError(err,false)
								//conn2.Close()
								sendMessageOverUDP(v, msg)
							}
							realReplicas=append(realReplicas,v)
							break
						}
					}
					filedata[fname]=FileMetaData{LatestVersion:filedata[fname].LatestVersion,ReplicaList:realReplicas}
				}
				//update filedata to no longer refer to failedID
				delete(nodedata,failedID)
				delete(unintroduced,failedID)
				if len(unintroduced) == 0 && !hasAllIntroduced {
					hasAllIntroduced = true
					mList := MembershipListReadAll()
					var cutoff int
					if len(mList)>=5 {
						cutoff=4
					} else {
						cutoff=len(mList)-1
					}
					for file,metadata := range filedata {
						if len(metadata.ReplicaList)<cutoff {
							replicaSet := make(map[string]bool)
							for _,v := range metadata.ReplicaList {
								replicaSet[v]=true
							}
							newReplicas:=make([]string,0)
							for _,v := range mList {
								if _,ok := replicaSet[v];!ok && v != INTRODUCER_ID {
									newReplicas=append(newReplicas,v)
									if (len(newReplicas)+len(metadata.ReplicaList))>=cutoff {
										break
									}
								}
							}
							payload := getJSONfromFsRemoteMessage(FsRemoteMessage{Cmd:0,SdfsFileName:file,Data:metadata.ReplicaList,LatestVersion:metadata.LatestVersion})
							message := createMessage(selfID,"FILE_GENERAL",string(payload),[]string{""})
							//msgJSON := getJSONfromMessage(message)
							for _,v := range newReplicas {
								//targetaddr,err := net.ResolveUDPAddr("udp",getIPfromID(v)+":"+getPortfromID(v))
								//ResolveError(err,true)
								//conn,err := net.DialUDP("udp",nil,targetaddr)
								//ResolveError(err,true)
								//_,err = conn.Write(msgJSON)
								//ResolveError(err,true)
								//conn.Close()
								sendMessageOverUDP(v, message)
								nodedata[v][file]=true
							}
							metadata.ReplicaList=append(metadata.ReplicaList,newReplicas...)
							filedata[file]=metadata
						}
					}
					//look at every file
					//if number of replicas of the file is < min(4,number of file storing nodes)
					//rebalance for that file
				}
				//now do job restart if necessary
				if _,ok := revboltallot[failedID]; ok {
					fmt.Println("A job node failed! Restarting...")
					boltallot=make(map[int]string)
					revboltallot=make(map[string][]int)
					jobRefCount = len(job.TopologyNodes)
					time.AfterFunc(6*time.Second,func() {
						mList := shuffle(MembershipListReadAll())
						mListIter := 0
						for _, tn := range job.TopologyNodes {
							for mList[mListIter] == INTRODUCER_ID || mList[mListIter] == MASTER_ID {
								mListIter++
								if mListIter==len(mList) {
									mListIter=0
								}
							}
							nodeID := mList[mListIter]
							boltallot[tn.ID]=nodeID
							if _,ok := revboltallot[nodeID]; ok {
								revboltallot[nodeID]=append(revboltallot[nodeID],tn.ID)
							} else {
								revboltallot[nodeID]=[]int{tn.ID}
							}

							mListIter++
						}
						fmt.Println("Assigned a job to following nodes:", boltallot)
						for _, tn := range job.TopologyNodes {
							//build the message to send that node here
							//allot job to a node
							tnm := convertTopologyNodeToTopologyNodesWithMachines(tn, boltallot)
							payload := string(getJSONfromTopologyNodeWithMachines(tnm))
							message := createMessage(selfID,"JOB_ALLOT",payload,[]string{""})
							sendMessageOverUDP(boltallot[tn.ID],message)
						}
						//master notifies hot standby
						estimMinID := "99999999999"
						for _, v := range mList {
							if v != INTRODUCER_ID && v != selfID {
								if v < estimMinID {
									estimMinID = v
								}
							}
						}
						//boltallot,revboltallot,job
						payload := ""
						standbyMsg := createMessage(selfID,"JOB_STANDBY",payload,[]string{""})
						sendMessageOverUDP(estimMinID,standbyMsg)
					})
				}
			} else if (failedID==MASTER_ID) {
				mList := MembershipListReadAll()
				minID := selfID
				for _,v := range mList {
					if v!=INTRODUCER_ID {
						if (v<minID) {
							minID = v
						}
					}
				}
				MASTER_ID = minID
				if selfID == MASTER_ID {
					fmt.Println("I am the new master node")
					hasAllIntroduced=false
					for _,v := range mList {
						if v!=INTRODUCER_ID {
							payload := FsRemoteMessage{Cmd:-200,SdfsFileName:"",Data:[]string{selfID},LatestVersion:0}
							msg := createMessage(selfID,"FILE_GENERAL",string(getJSONfromFsRemoteMessage(payload)),[]string{""})
							//targetaddr,err := net.ResolveUDPAddr("udp",getIPfromID(v)+":"+getPortfromID(v))
							//ResolveError(err,true)
							//conn,err := net.DialUDP("udp",nil,targetaddr)
							//ResolveError(err,false)
							//_,err = conn.Write(getJSONfromMessage(msg))
							//ResolveError(err,false)
							//conn.Close()
							sendMessageOverUDP(v, msg)
							unintroduced[v]=true
						}
					}
				}
			} //else do nothing
		case newID := <-masternchan:
			if (MASTER_ID==selfID) {
				nodedata[newID]=make(map[string]bool)
				for fname,mdata := range filedata {
					if len(mdata.ReplicaList)<4 {
						payload:=FsRemoteMessage{Cmd:0,SdfsFileName:fname,Data:mdata.ReplicaList,LatestVersion:filedata[fname].LatestVersion}
						msg := createMessage(selfID,"FILE_GENERAL",string(getJSONfromFsRemoteMessage(payload)),[]string{""})
						{
							//targetaddr,err := net.ResolveUDPAddr("udp",getIPfromID(newID) + ":"+ getPortfromID(newID))
							//ResolveError(err,false)
							//conn2,err := net.DialUDP("udp",nil,targetaddr)
							//msgJSON := getJSONfromMessage(msg)
							//_, err = conn2.Write(msgJSON)
							//ResolveError(err,false)
							//conn2.Close()
							sendMessageOverUDP(newID, msg)
						}
						mdata.ReplicaList=append(mdata.ReplicaList,newID)
						filedata[fname]=FileMetaData{LatestVersion:filedata[fname].LatestVersion,ReplicaList:mdata.ReplicaList}
					}
				}
			}
		}
	}
}


func TCPFileServer(serv net.Listener,realfilepath string, numberOfReplicas int) {
	for i := 0 ; i != numberOfReplicas; i++ {
		rdfile,err:=os.Open(realfilepath)
		ResolveError(err,false)
		destsocket,err:=serv.Accept()
		ResolveError(err,false)
		io.Copy(destsocket,rdfile)
		destsocket.Close()
	}
	serv.Close()
	time_elapsed := time.Since(startTime)
	fmt.Println("Time :", time_elapsed)
}

func FileSystem() {
	err := os.RemoveAll(sdfsDirName+"/")
	ResolveError(err,true)
	os.Mkdir(sdfsDirName,0755)
	sdfsfilenamecache:=""
	localfilenamecache:=""
	localnumversions:=0
	sdfsfiledir := make(map[string]int)
	for {
		select {
		case order:= <-fschan:
			switch order.Cmd {
			case 0: //put
				startTime = time.Now()
				masterMessage := MasterMessage{Cmd: 0, SdfsFileName:order.SdfsFileName}
				payload := string(getJSONfromMasterMessage(masterMessage))
				msg:=createMessage(selfID,"FILE_MASTER",payload,[]string{"NONE"})
				//send message to master to get new version number, and replica targets
				localfilenamecache=order.LocalFileName
				sdfsfilenamecache=order.SdfsFileName
				//cache the local file name
				{
					//target,err := net.ResolveUDPAddr("udp",getIPfromID(MASTER_ID)+":"+getPortfromID(MASTER_ID))
					//ResolveError(err,false)
					//conn,err := net.DialUDP("udp",nil,target)
					//ResolveError(err,false)
					//_,err = conn.Write(getJSONfromMessage(msg))
					//ResolveError(err,false)
					//conn.Close()
					sendMessageOverUDP(MASTER_ID, msg)
				}
				//send the message here
			case 1: //get
				//send message to master to find latest version number, and file source
				startTime = time.Now()
				masterMessage := MasterMessage{Cmd: 1, SdfsFileName:order.SdfsFileName}
				payload := string(getJSONfromMasterMessage(masterMessage))
				msg:=createMessage(selfID,"FILE_MASTER",payload,[]string{"NONE"})
				//send message to master to get new version number, and replica targets
				//cache the local file name
				localfilenamecache=order.LocalFileName
				{
					//target,err := net.ResolveUDPAddr("udp",getIPfromID(MASTER_ID)+":"+getPortfromID(MASTER_ID))
					//ResolveError(err,false)
					//conn,err := net.DialUDP("udp",nil,target)
					//ResolveError(err,false)
					//_,err = conn.Write(getJSONfromMessage(msg))
					//ResolveError(err,false)
					//conn.Close()
					sendMessageOverUDP(MASTER_ID, msg)
				}
				//if you are a replica with latest version, just copy
				//else ask a replica
			case 2: //delete
				//send message to master to order delete
				//wait for ack
				startTime = time.Now()
				masterMessage := MasterMessage{Cmd: 2,SdfsFileName:order.SdfsFileName}
				payload := string(getJSONfromMasterMessage(masterMessage))
				msg := createMessage(selfID,"FILE_MASTER",payload,[]string{"None"})
				{
					//target,err := net.ResolveUDPAddr("udp",getIPfromID(MASTER_ID)+":"+getPortfromID(MASTER_ID))
					//ResolveError(err,false)
					//conn,err := net.DialUDP("udp",nil,target)
					//ResolveError(err,false)
					//_,err = conn.Write(getJSONfromMessage(msg))
					//ResolveError(err,false)
					//conn.Close()
					sendMessageOverUDP(MASTER_ID, msg)
				}
			case 3: //ls
				//send message to master to get files and Data
				//wait for response and print that out
				masterMessage := MasterMessage{Cmd: 3, SdfsFileName:order.SdfsFileName}
				payload := string(getJSONfromMasterMessage(masterMessage))
				msg:=createMessage(selfID,"FILE_MASTER",payload,[]string{"NONE"})
				sdfsfilenamecache=order.SdfsFileName
				{
					//target,err := net.ResolveUDPAddr("udp",getIPfromID(MASTER_ID)+":"+getPortfromID(MASTER_ID))
					//ResolveError(err,false)
					//conn,err := net.DialUDP("udp",nil,target)
					//ResolveError(err,false)
					//_,err = conn.Write(getJSONfromMessage(msg))
					//ResolveError(err,false)
					//conn.Close()
					sendMessageOverUDP(MASTER_ID, msg)
				}
			case 4: //store
				//compile local list of files
				//Use SdfsFileDir to do this
				fmt.Println("Machine:" + selfID + " is storing:")

				for f, v := range sdfsfiledir {
					for i := 1; i <= v; i++ {
						fmt.Println("File: ",f, ", Version: ",i)
					}
				}

			case 5: //get-versions
				//contact master
				//show responses
				//send message to master to find latest version number, and file source
				startTime = time.Now()
				masterMessage := MasterMessage{Cmd: 5, SdfsFileName:order.SdfsFileName}
				payload := string(getJSONfromMasterMessage(masterMessage))
				msg:=createMessage(selfID,"FILE_MASTER",payload,[]string{"NONE"})
				localfilenamecache=order.LocalFileName
				localnumversions = order.NumVersions
				{
					//target,err := net.ResolveUDPAddr("udp",getIPfromID(MASTER_ID)+":"+getPortfromID(MASTER_ID))
					//ResolveError(err,false)
					//conn,err := net.DialUDP("udp",nil,target)
					//ResolveError(err,false)
					//_,err = conn.Write(getJSONfromMessage(msg))
					//ResolveError(err,false)
					//conn.Close()
					sendMessageOverUDP(MASTER_ID, msg)
				}
			}
			// DEALING WITH RESPONSE FROM MASTER VS EACH OTHER case
		case fsmsg := <-fsmsgchan:
			switch(fsmsg.Cmd) {
			case -200:
				jsonSdfsFileDir, err := json.Marshal(sdfsfiledir)
				ResolveError(err, false)
				payload := string(getJSONfromMasterMessage(MasterMessage{Cmd:-1, SdfsFileName: string(jsonSdfsFileDir)}))
				msg := createMessage(selfID, "FILE_INTRO",payload,[]string{""})
				if MASTER_ID > fsmsg.Data[0] {
					MASTER_ID = fsmsg.Data[0]
				}
				{
					//target,err := net.ResolveUDPAddr("udp",getIPfromID(fsmsg.Data[0])+":"+getPortfromID(fsmsg.Data[0]))
					//ResolveError(err,false)
					//conn,err := net.DialUDP("udp",nil,target)
					//ResolveError(err,false)
					//_,err = conn.Write(getJSONfromMessage(msg))
					//ResolveError(err,false)
					sendMessageOverUDP(fsmsg.Data[0], msg)
				}
			case 0:
				servsocket,err := net.Listen("tcp",":0")
				ResolveError(err,true)
				targetIdx := 0
				replicas := shuffle(fsmsg.Data)
				targetID := replicas[targetIdx] //need to actually dynamically adjust target
				addr_string := servsocket.Addr().String()
				addr_split := strings.Split(addr_string, ":")
				port_num := addr_split[len(addr_split)-1]
				target_to_send_to := getIPfromID(selfID) + ":"+ port_num
				//fmt.Println("Target: ", target_to_send_to)
				for i := 1 ; i <= fsmsg.LatestVersion ; i++ {
					mkfile, err := os.Create(SDFSToReal(fsmsg.SdfsFileName, i))
					ResolveError(err, true)
					payload := string(getJSONfromFsRemoteMessage(FsRemoteMessage{Cmd: 2,
						SdfsFileName:  fsmsg.SdfsFileName,
						Data:          []string{target_to_send_to},
						LatestVersion: i}))
					msg := createMessage(selfID, "FILE_GENERAL", payload, []string{""})
					{
						//target, err := net.ResolveUDPAddr("udp", getIPfromID(targetID)+":"+getPortfromID(targetID))
						//ResolveError(err, false)
						//conn, err := net.DialUDP("udp", nil, target)
						//ResolveError(err, false)
						//_, err = conn.Write(getJSONfromMessage(msg))
						//ResolveError(err, false)
						sendMessageOverUDP(targetID, msg)
					}
					readconn, err := servsocket.Accept()
					ResolveError(err, true)
					_, err = io.Copy(mkfile, readconn)
					if err != nil {
						i--
						targetIdx++
						targetID = replicas[targetIdx]
					}
					mkfile.Close()
					readconn.Close()
				}
			case -1: //message from master, has replicas attached
				servsocket,err := net.Listen("tcp",":0")
				ResolveError(err,true)
				go TCPFileServer(servsocket,localfilenamecache,len(fsmsg.Data))
				addr_string := servsocket.Addr().String()
				addr_split := strings.Split(addr_string, ":")
				port_num := addr_split[len(addr_split)-1]
				target_to_send_to := getIPfromID(selfID) + ":"+ port_num
				for _,repID := range fsmsg.Data {
					//send replica Data to these nodes as message type 1
					payload := string(getJSONfromFsRemoteMessage(FsRemoteMessage{Cmd: 1,
						SdfsFileName: sdfsfilenamecache,
						Data:         []string{target_to_send_to},
						LatestVersion: fsmsg.LatestVersion}))
					msg := createMessage(selfID, "FILE_GENERAL", payload, []string{""})
					//target,err := net.ResolveUDPAddr("udp",getIPfromID(repID)+":"+getPortfromID(repID))
					//ResolveError(err,false)
					//conn,err := net.DialUDP("udp",nil,target)
					//ResolveError(err,false)
					//_,err = conn.Write(getJSONfromMessage(msg))
					//ResolveError(err,false)
					sendMessageOverUDP(repID, msg)
				}
			case -4:
				replicas := fsmsg.Data
				fmt.Println("The Replicas where " + fsmsg.SdfsFileName + " are stored are:")
				for _, eachReplica := range replicas{
					fmt.Println(eachReplica)
				}
				fmt.Println("The Latest Version is " + strconv.Itoa(fsmsg.LatestVersion))
			case -400:
				fmt.Println("The File " + fsmsg.SdfsFileName + " does not exist in SDFS!")
			case 1:
				sdfsfiledir[fsmsg.SdfsFileName]=fsmsg.LatestVersion
				mkfile,err:=os.Create(SDFSToReal(fsmsg.SdfsFileName, fsmsg.LatestVersion))
				ResolveError(err,true)
				addr,err:=net.ResolveTCPAddr("tcp",fsmsg.Data[0])
				ResolveError(err,true)
				conn,err:=net.DialTCP("tcp",nil,addr)
				ResolveError(err,true)
				io.Copy(mkfile,conn)
				mkfile.Close()
			case 2:
				rdfile,err:=os.Open(SDFSToReal(fsmsg.SdfsFileName,fsmsg.LatestVersion))
				ResolveError(err,true)
				writeconn,err:=net.Dial("tcp",fsmsg.Data[0])
				ResolveError(err,true)
				io.Copy(writeconn,rdfile)
				rdfile.Close()
				writeconn.Close()
			case -2:
				servsocket,err := net.Listen("tcp",":0")
				ResolveError(err,true)
				addr_string := servsocket.Addr().String()
				addr_split := strings.Split(addr_string, ":")
				port_num := addr_split[len(addr_split)-1]
				target_to_send_to := getIPfromID(selfID) + ":"+ port_num
				//fmt.Println("Target: ", target_to_send_to)
				mkfile,err:=os.Create(localfilenamecache)
				ResolveError(err,true)
				payload := string(getJSONfromFsRemoteMessage(FsRemoteMessage{Cmd: 2,
					SdfsFileName: fsmsg.SdfsFileName,
					Data: []string{target_to_send_to},
					LatestVersion: fsmsg.LatestVersion}))
				msg := createMessage(selfID, "FILE_GENERAL",payload,[]string{""})
				{
					//target,err := net.ResolveUDPAddr("udp",getIPfromID(fsmsg.Data[0])+":"+getPortfromID(fsmsg.Data[0]))
					//ResolveError(err,false)
					//conn,err := net.DialUDP("udp",nil,target)
					//ResolveError(err,false)
					//_,err = conn.Write(getJSONfromMessage(msg))
					//ResolveError(err,false)
					sendMessageOverUDP(fsmsg.Data[0], msg)
				}
				readconn,err:=servsocket.Accept()
				ResolveError(err,true)
				io.Copy(mkfile,readconn)
				mkfile.Close()
				readconn.Close()
				elapsed_time := time.Since(startTime)
				fmt.Println("Elapsed time: ",elapsed_time)
			case 5:
				rdfile,err:=os.Open(SDFSToReal(fsmsg.SdfsFileName,fsmsg.LatestVersion))
				ResolveError(err,true)
				writeconn,err:=net.Dial("tcp",fsmsg.Data[0])
				ResolveError(err,true)
				io.Copy(writeconn,rdfile)
				rdfile.Close()
				writeconn.Close()
			case -5:
				sdfsLatestVersion := fsmsg.LatestVersion
				data := ""
				mkfile,err:=os.Create(localfilenamecache)
				ResolveError(err, true)
				for i:= sdfsLatestVersion; i > (sdfsLatestVersion - localnumversions); i--{
					data = "\n\nVersion" + strconv.Itoa(i) + "\n-----------------------\n"
					servsocket,err := net.Listen("tcp",":0")
					ResolveError(err,true)
					addr_string := servsocket.Addr().String()
					addr_split := strings.Split(addr_string, ":")
					port_num := addr_split[len(addr_split)-1]
					target_to_send_to := getIPfromID(selfID) + ":"+ port_num

					ResolveError(err,true)
					payload := string(getJSONfromFsRemoteMessage(FsRemoteMessage{Cmd: 5,
						SdfsFileName: fsmsg.SdfsFileName,
						Data: []string{target_to_send_to},
						LatestVersion: i}))
					msg := createMessage(selfID, "FILE_GENERAL",payload,[]string{""})
					{
						//target,err := net.ResolveUDPAddr("udp",getIPfromID(fsmsg.Data[0])+":"+getPortfromID(fsmsg.Data[0]))
						//ResolveError(err,false)
						//conn,err := net.DialUDP("udp",nil,target)
						//ResolveError(err,false)
						//_,err = conn.Write(getJSONfromMessage(msg))
						//ResolveError(err,false)
						sendMessageOverUDP(fsmsg.Data[0], msg)
					}
					readconn,err:=servsocket.Accept()
					ResolveError(err,true)
					mkfile.WriteString(data)
					io.Copy(mkfile,readconn)
					readconn.Close()
				}
				mkfile.Close()
				elapsed_time := time.Since(startTime)
				fmt.Println("Elapsed time: ",elapsed_time)

			case -3:
				for _,repID := range fsmsg.Data {
					//send replica Data to these nodes as message type 1
					payload := string(getJSONfromFsRemoteMessage(FsRemoteMessage{Cmd: 3,
						SdfsFileName: fsmsg.SdfsFileName,
						Data:         []string{},
						LatestVersion: fsmsg.LatestVersion}))
					msg := createMessage(selfID, "FILE_GENERAL", payload, []string{""})
					//target,err := net.ResolveUDPAddr("udp",getIPfromID(repID)+":"+getPortfromID(repID))
					//ResolveError(err,false)
					//conn,err := net.DialUDP("udp",nil,target)
					//ResolveError(err,false)
					//_,err = conn.Write(getJSONfromMessage(msg))
					//ResolveError(err,false)
					sendMessageOverUDP(repID, msg)
				}
				elapsed_time := time.Since(startTime)
				fmt.Println("Elapsed time:",elapsed_time)
			case 3:
				for i := 1 ; i <= fsmsg.LatestVersion; i++ {
					err := os.Remove(SDFSToReal(fsmsg.SdfsFileName,i))
					ResolveError(err,false)
				}
				delete(sdfsfiledir,fsmsg.SdfsFileName)
				delete(sdfsfiledir,fsmsg.SdfsFileName)
			}

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
	//1log.Println("Got Join Response From:" + respMessage.ID )
	//1log.Println(string(respJSON))
	//Populate Membership List from Response
	if respMessage.Type == "JOINACK" {
		MASTER_ID = respMessage.Payload
		INTRODUCER_ID = respMessage.ID
		if (selfID == MASTER_ID) {
			fmt.Println(" I am the Master")
		}
		go MasterServer()
		go JobSystem()
		//END
		membershipList := respMessage.AdditionalData
		for _,v := range membershipList {
			MembershipListInsert(v)
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
	MembershipListInsert(selfID)

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
		//1log.Println("Got:" + string(respJSON))
		if err!=nil{
			//INCASE OF TIMEOUT CONTINUE
			continue
		} else {
			//INCASE OF INTRODUCEACK, ADD TO MEMBERSHIP LIST
			responseMessage := getMessagefromJSON(respJSON)
			MembershipListInsert(responseMessage.ID)
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
			//1log.Println("Got an Introduce Ack:" + string(joinRequestJSON))
		} else {
			introduceMessage := createMessage(selfID, "INTRODUCE", joinRequestMessage.ID, []string{"None"})
			membershipList := MembershipListReadAll()
			for _, member := range membershipList {
				member_IP := getIPfromID(member)
				member_Port := getPortfromID(member)
				if member_IP == selfIPAddress && member_Port == strconv.Itoa(serverPortNumber) {
					continue
				}

				//REFAC
				//clientaddr, err := net.ResolveUDPAddr("udp", member_IP+":"+member_Port)
				//ResolveError(err,false)
				//_, err = serv.WriteToUDP(introduceMessageJSON, clientaddr)
				//ResolveError(err,false)
				sendMessageOverUDP(member, introduceMessage)
			}

			//update own membership list here
			MembershipListInsert(joinRequestMessage.ID)
			membershipList = MembershipListReadAll()
			if len(membershipList)==2 {
				MASTER_ID=joinRequestMessage.ID
			}
			//respond to join request
			joinResponse := createMessage(selfID, "JOINACK", MASTER_ID, membershipList)
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
			//1log.Println("Received: " + string(resp) + " From: " + targetID )
			if err!= nil {
				ticker.Stop()
				stopflag = true

				//need to notify self ping server
				//REFAC
				//clientaddr, err := net.ResolveUDPAddr("udp",getIPfromID(selfID)+":"+getPortfromID(selfID))
				//ResolveError(err,false)
				//conn2, err := net.DialUDP("udp",nil,clientaddr)
				//ResolveError(err,false)
				//Send the failure messagee
				failMessage := createMessage(selfID, "FAILURE", targetID , []string{"None"})
				//failJSON := getJSONfromMessage(failMessage)
				//_, err = conn2.Write(failJSON)
				//ResolveError(err,false)
				//conn2.Close()
				sendMessageOverUDP(selfID, failMessage)
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
	for {
		n,addr, err := serv.ReadFromUDP(buf)
		messageJSON := []byte(string(buf[:n]))
		message := getMessagefromJSON(messageJSON)
		//buf now has the Data that was sent
		ResolveError(err,false)
		//go Ack(addr)
		if message.Type == "PING" {
			ackMessage := createMessage(selfID, "ACK", "Nothing A", []string{"None"})
			_, err = serv.WriteToUDP(getJSONfromMessage(ackMessage), addr)
			ResolveError(err,false)
		} else if message.Type == "INTRODUCE" {
			MembershipListInsert(message.Payload)
			refocus<-1
			if selfID == MASTER_ID {
				masternchan <- message.Payload
			}
			ackMessage := createMessage(selfID, "INTRODUCEACK", "Nothing", []string{"None"})
			_, err = serv.WriteToUDP(getJSONfromMessage(ackMessage), addr)
			ResolveError(err,false)
		} else if message.Type == "FAILURE" || message.Type == "LEAVE" {
			result:= MembershipListDelete(message.Payload)
			if result[0]=="TRUE" { //need to send failure info to neighbours
				if message.Type == "FAILURE" {
					fmt.Println("Node ", message.Payload , " has failed")
				} else if message.Type == "LEAVE"{
					fmt.Println("Node ", message.Payload , " has left")
				}
				if selfID!=INTRODUCER_ID {
					masterfchan <- message.Payload
				} else {
					if message.Payload == MASTER_ID {
						minID := "9999999999999999999"
						mList := MembershipListReadAll()
						for _,v := range mList {
							if v!=INTRODUCER_ID {
								if (v<minID) {
									minID = v
								}
							}
						}
						MASTER_ID = minID
					}
				}
				mList:= MembershipListReadAll()
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
							//REFAC
							//clientaddr, err := net.ResolveUDPAddr("udp",getIPfromID(v) + ":"+ getPortfromID(v))
							//ResolveError(err,false)
							//conn2, err := net.DialUDP("udp",nil,clientaddr)
							////Send the failure messagee
							failMessage := createMessage(selfID, message.Type, message.Payload , []string{"None"})
							//failJSON := getJSONfromMessage(failMessage)
							//_, err = conn2.Write(failJSON)
							//ResolveError(err,false)
							//conn2.Close()
							sendMessageOverUDP(v, failMessage)
						}
					}
				} else { //need to send to 3 neighbours
					targets:=make([]string,3)
					targets[0]=mList[(selfidx-2+len(mList))%len(mList)]
					targets[1]=mList[(selfidx-1+len(mList))%len(mList)]
					targets[2]=mList[(selfidx+1+len(mList))%len(mList)]
					for _,v := range targets {
						//REFAC
						//clientaddr, err := net.ResolveUDPAddr("udp",getIPfromID(v) + ":"+ getPortfromID(v))
						//ResolveError(err,false)
						//conn3, err := net.DialUDP("udp",nil,clientaddr)
						////send a message indicating a failure here
						failMessage := createMessage(selfID, message.Type, message.Payload , []string{"None"})
						//failJSON := getJSONfromMessage(failMessage)
						//_, err = conn3.Write(failJSON)
						//ResolveError(err,false)
						//conn3.Close()
						sendMessageOverUDP(v, failMessage)
					}
				}
				refocus<- -1
			}
		} else if message.Type == "FILE_GENERAL" {
			fsmsgchan <- getFsRemoteMessagefromJSON([]byte(message.Payload))

			//take payload
			//convert payload to FsRemoteMessage
			//send it along fsmsgchan

		} else if message.Type == "FILE_MASTER" {
			//take payload
			mmsg := getMasterMessagefromJSON([]byte(message.Payload))
			packet := MasterPacket{ID:message.ID, Cmd:mmsg.Cmd, SdfsFileName:mmsg.SdfsFileName, NumVersions:0}
			masterichan<-packet
			response:= <-masterochan
			replyMessage := createMessage(selfID,"FILE_GENERAL",string(getJSONfromFsRemoteMessage(response)),[]string{"None"})
			{
				//target,err := net.ResolveUDPAddr("udp",getIPfromID(message.ID)+":"+getPortfromID(message.ID))
				//ResolveError(err,true)
				//conn,err := net.DialUDP("udp",nil,target)
				//ResolveError(err,true)
				//conn.Write(getJSONfromMessage(replyMessage))
				//conn.Close()
				sendMessageOverUDP(message.ID, replyMessage)
			}
			//send message as response
			//convert payload to MasterPacket
			//send it along masterichan
			/*
			 *   { operation: PUT/GET/... , SdfsFileName: SDFSFILENAME }
			 */
			//read from masterochan
		} else if message.Type == "FILE_INTRO"{
			mmsg := getMasterMessagefromJSON([]byte(message.Payload))
			packet := MasterPacket{ID:message.ID, Cmd:mmsg.Cmd, SdfsFileName:mmsg.SdfsFileName, NumVersions:0}
			masterichan<-packet
		} else if message.Type == "JOB_CREATE" {
			//nodes := ConvertNodesDataToNodes(message.Payload)
			nodes := message.Payload
			masterjchan<-nodes
		} else if message.Type == "JOB_ALLOT" {
			msg := getTopologyNodeWithMachinesfromJSON([]byte(message.Payload))
			jobchan <- msg
		} else if message.Type == "JOB_CONTACT" {
			//msg := getContactMessagefromJSON([]byte(message.Payload))
			msg := ContactMessage{SrcID:message.ID,Addr:message.Payload}
			srcchan<-msg
		} else if message.Type == "JOB_DONE" {
			masterjobdonechan<-message.ID
		} else if message.Type == "JOB_STANDBY" {
			masterschan <- message.Payload
		}
	}
}

var masterjobdonechan = make(chan string,100)

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
			mList:= MembershipListReadAll()
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
				targets:=make([]string,3)
				targets[0]=mList[(selfidx-2+len(mList))%len(mList)]
				targets[1]=mList[(selfidx-1+len(mList))%len(mList)]
				targets[2]=mList[(selfidx+1+len(mList))%len(mList)]
				for _,v := range targets {
					clientaddr, err := net.ResolveUDPAddr("udp",getIPfromID(v) + ":"+ getPortfromID(v))
					ResolveError(err,false)
					go PingClient(clientaddr,v)
				}
			}
		}
	}
}
