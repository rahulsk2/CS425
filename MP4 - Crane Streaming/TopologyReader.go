package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

type TopologyNodes struct {
	TopologyNodes []TopologyNode `json:"topology_nodes"`
}

type TopologyNode struct {
	ID   int `json:"ID"`
	OpType   string `json:"OpType"`
	FileToExecute   string `json:"FileToExecute"`
	InputBolts []int `json:"InputBolts"`
	OutputBolts []int `json:"OutputBolts"`
	MethodToExecute    int    `json:"MethodToExecute"`
	FileIO   string `json:"FileIO"`
}

type TopologyNodeWithMachines struct {
	ID   int
	MachineID string
	OpType   string
	FileToExecute   string
	InputBolts []string
	OutputBolts []string
	MethodToExecute    int
	FileIO   string
}

func convertTopologyNodeToTopologyNodesWithMachines(node TopologyNode, boltMap map[int]string) TopologyNodeWithMachines{
	var tnm TopologyNodeWithMachines
	tnm.ID = node.ID
	tnm.MachineID = boltMap[node.ID]
	tnm.OpType = node.OpType
	tnm.FileToExecute = node.FileToExecute
	for _, v := range node.InputBolts {
		tnm.InputBolts = append(tnm.InputBolts, boltMap[v])
	}
	for _, v := range node.OutputBolts {
		tnm.OutputBolts = append(tnm.OutputBolts, boltMap[v])
	}
	tnm.MethodToExecute = node.MethodToExecute
	tnm.FileIO = node.FileIO
	return tnm
}


func ReadTopology(fileName string) TopologyNodes {
	jsonFile, err := os.Open(fileName)
	ResolveError(err, true)
	fmt.Println("Successfully Opened Topology")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var nodes TopologyNodes
	err = json.Unmarshal(byteValue, &nodes)
	ResolveError(err, true)
	return nodes

	//for i := 0; i < len(nodes.TopologyNodes); i++ {
	//	fmt.Println("\n\nID: " + strconv.Itoa(nodes.TopologyNodes[i].ID))
	//	fmt.Println("OpType: " + nodes.TopologyNodes[i].OpType)
	//	fmt.Println("FileToExecute: " + nodes.TopologyNodes[i].FileToExecute)
	//	fmt.Println("InputBolts:", nodes.TopologyNodes[i].InputBolts)
	//	//for j := 0 ; j < len(nodes.TopologyNodes[i].InputBolts) ; j++ {
	//	//	fmt.Println("IP Bolt: " + nodes.TopologyNodes[i].InputBolts[j])
	//	//}
	//	fmt.Println("OutputBolts:" , nodes.TopologyNodes[i].OutputBolts)
	//	//for j := 0 ; j < len(nodes.TopologyNodes[i].OutputBolts) ; j++ {
	//	//	fmt.Println("OP Bolt: " + nodes.TopologyNodes[i].OutputBolts[j])
	//	//}
	//	fmt.Println("Method To Execute: " + strconv.Itoa(nodes.TopologyNodes[i].MethodToExecute))
	//	fmt.Println("FileIO: " + nodes.TopologyNodes[i].FileIO)
	//}

}

func ConvertNodesDataToNodes(nodesData string) TopologyNodes{
	var nodes TopologyNodes
	err := json.Unmarshal([]byte(nodesData), &nodes)
	ResolveError(err, true)
	return nodes
}

func ReadTopologyFile(fileName string) string {
	jsonFile, err := os.Open(fileName)
	ResolveError(err, true)
	fmt.Println("Successfully Opened Topology")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	return string(byteValue)
}

func getJSONfromTopologyNode(node TopologyNode) []byte {
	var jsonMessage []byte
	jsonMessage, err := json.Marshal(node)
	if err != nil {
		log.Println(err)
	}
	return jsonMessage
}

func getTopologyNodefromJSON(jsonNode []byte) TopologyNode {
	var node TopologyNode
	err := json.Unmarshal(jsonNode, &node)
	if err != nil {
		log.Println(err)
	}
	return node
}

func getJSONfromTopologyNodes(node TopologyNodes) []byte {
	var jsonMessage []byte
	jsonMessage, err := json.Marshal(node)
	if err != nil {
		log.Println(err)
	}
	return jsonMessage
}

func getTopologyNodesfromJSON(jsonNode []byte) TopologyNodes {
	var node TopologyNodes
	err := json.Unmarshal(jsonNode, &node)
	if err != nil {
		log.Println(err)
	}
	return node
}

func getJSONfromTopologyNodeWithMachines(node TopologyNodeWithMachines) []byte {
	var jsonMessage []byte
	jsonMessage, err := json.Marshal(node)
	if err != nil {
		log.Println(err)
	}
	return jsonMessage
}

func getTopologyNodeWithMachinesfromJSON(jsonNode []byte) TopologyNodeWithMachines {
	var node TopologyNodeWithMachines
	err := json.Unmarshal(jsonNode, &node)
	if err != nil {
		log.Println(err)
	}
	return node
}