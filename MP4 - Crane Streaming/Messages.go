package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

//MAIN MESSAGE
type Message struct {
	ID    string
	Type string
	Payload string
	AdditionalData []string
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
		fmt.Println(string(jsonMessage))
		log.Println(err)
	}
	return message
}



//TOPOLOGY MESSAGE

type TopologyMessage struct {
	InputBolts      []string
	OutputBolts     []string
	FileToExecute   string
	MethodToExecute int
	OpType          string //SPOUT, FILTER, JOIN, TRANSFORM, SINK??
	FileIO          string
}

func makeCopy(message TopologyMessage) TopologyMessage {
	neo := TopologyMessage{InputBolts:make([]string,len(message.InputBolts)),OutputBolts:make([]string,len(message.OutputBolts)),FileToExecute:message.FileToExecute,MethodToExecute:message.MethodToExecute,OpType:message.OpType, FileIO:message.FileIO}
	copy(neo.InputBolts,message.InputBolts)
	copy(neo.OutputBolts,message.OutputBolts)
	return neo
}

func createTopologyMessage(InputBolts    []string, OutputBolts []string, FileToExecute string, MethodToExecute int, OpType string, FileForJoin string) TopologyMessage {
	message := TopologyMessage{
		InputBolts :      InputBolts,
		OutputBolts :     OutputBolts,
		FileToExecute :   FileToExecute,
		MethodToExecute : MethodToExecute,
		OpType :          OpType,
		FileIO:           FileForJoin,
	}
	return message
}

func getJSONfromTopologyMessage(message TopologyMessage) []byte {
	var jsonMessage []byte
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
	}
	return jsonMessage
}

func getTopologyMessagefromJSON(jsonMessage []byte) TopologyMessage {
	var message TopologyMessage
	err := json.Unmarshal(jsonMessage, &message)
	if err != nil {
		log.Println(err)
	}
	return message
}


// Contact Message

type ContactMessage struct {
	SrcID string
	Addr string
}

func getJSONfromContactMessage(message ContactMessage) []byte {
	var jsonMessage []byte
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
	}
	return jsonMessage
}

func getContactMessagefromJSON(jsonMessage []byte) ContactMessage {
	var message ContactMessage
	err := json.Unmarshal(jsonMessage, &message)
	if err != nil {
		log.Println(err)
	}
	return message
}



//MISC

type FsCommand struct {
	Cmd           int //0 - put, 1 - get, 2 - delete, 3 - ls, 5 - store
	SdfsFileName  string
	LocalFileName string
	NumVersions int
}

type MasterMessage struct {
	Cmd          int //put, get, delete, ls
	SdfsFileName string
}

type FileMetaData struct {
	LatestVersion int
	ReplicaList   []string
}

type MasterPacket struct {
	ID           string
	Cmd          int //0 - put, 1 - get , 2 - delete, 3 - ls, 4 - failure/leave, 5 - get-versions
	SdfsFileName string
	NumVersions  int
}

type FsRemoteMessage struct {
	Cmd           int //0 - replica list attached, 1 - put (file attached), 2 - get (send file in response), 3 - delete (filename attached)
	SdfsFileName  string
	Data          []string
	LatestVersion int
}


func getMapfromJSON(jsonMessage []byte) map[string]int {
	var filenamemap map[string]int
	err := json.Unmarshal(jsonMessage, &filenamemap)
	if err != nil {
		log.Println(err)
	}
	return filenamemap
}

func getMapIntStringfromJSON(jsonMessage []byte) map[int]string {
	var filenamemap map[int]string
	err := json.Unmarshal(jsonMessage, &filenamemap)
	if err != nil {
		log.Println(err)
	}
	return filenamemap
}

func getJSONfromMasterMessage(message MasterMessage) []byte {
	var jsonMessage []byte
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return jsonMessage
}

func getMasterMessagefromJSON(jsonMessage []byte) MasterMessage {
	var message MasterMessage
	err := json.Unmarshal(jsonMessage, &message)
	if err != nil {
		fmt.Println(err)
	}
	return message
}

func getJSONfromFsRemoteMessage(message FsRemoteMessage) []byte {
	var jsonMessage []byte
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
	}
	return jsonMessage
}

func getFsRemoteMessagefromJSON(jsonMessage []byte) FsRemoteMessage {
	var message FsRemoteMessage
	err := json.Unmarshal(jsonMessage, &message)
	if err != nil {
		log.Println(err)
	}
	return message
}