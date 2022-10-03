package configuration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type ConfigNode struct {
	Topic        			 string
	Gossip					 bool
	ConnectionProbability	 float32
}



type commonConfigFields struct{
	Nodecount     			 int          			`json:"Nodecount"`
	GossipHeaders 			 bool         			`json:"GossipHeaders"`
	ConnectionProbability	 float32				`json:"ConnectionProbability"`
	Topic        			 string     			`json:"Topic"`

}

type MeshConfig struct {
	commonConfigFields
	Nodes         			 []ConfigNode
}





func (c *MeshConfig) generateNodes(){
	for i := 0;i<c.Nodecount;i++{
		c.Nodes = append(c.Nodes, ConfigNode{})
		c.Nodes[i].Gossip = c.GossipHeaders
		c.Nodes[i].ConnectionProbability = c.ConnectionProbability
	}
}

func loadJson(jsonfilename string) (*commonConfigFields, error) {

	fileinfo, err := os.Stat(jsonfilename)
	if err != nil{return nil, err}
	fmt.Print("\nloading configuration from:",jsonfilename," modified:",fileinfo.ModTime().Local())

	file, err := os.OpenFile(jsonfilename,os.O_RDONLY,0666)
	if err != nil{return nil,err}
	
	data, err := ioutil.ReadAll(file)
	if err != nil{return nil,err}

	config := new(commonConfigFields)
	err = json.Unmarshal(data,config)
	if err != nil{return nil,err}

	return config,nil
}


func NewMeshConfig(jsonfilename string) (*MeshConfig,error) {

	cfg, err := loadJson(jsonfilename)
	if err != nil{return nil,err}

	configuration := &MeshConfig{
		Nodes:         []ConfigNode{},
		commonConfigFields:*cfg,
		
	}

	configuration.generateNodes()
	
	return configuration,nil
}


func (c *MeshConfig) Print(){

	data,err := json.MarshalIndent(c,"","  ")
	if err == nil{
		fmt.Print(string(data))
		return
	}
}

