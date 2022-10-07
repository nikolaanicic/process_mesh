package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mesh/configuration"
	"net/http"
	"os"
	"os/exec"
)



const coordinatorpath = "http://192.168.9.101"

func signup(w http.ResponseWriter,req *http.Request){

	defer req.Body.Close()
	http.Post(coordinatorpath + "/signup","application/text",req.Body)
	
}

func startnexttest(w http.ResponseWriter, req *http.Request){
	configbytes, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil{return}

	var config configuration.MeshConfig
	if err := json.Unmarshal(configbytes,&config); err != nil{return}

	go startnodes(config)
}


func handleresults(w http.ResponseWriter, req *http.Request){
	http.Post(coordinatorpath + "/results","application/json",req.Body)
	defer req.Body.Close()

}

func isdone(w http.ResponseWriter, req *http.Request){
	os.Exit(0)
}



func startnodes(config configuration.MeshConfig){


	fmt.Println("starting the nodes")
	for _,confignode := range config.Nodes{
		
		cmdpath, err := exec.LookPath("cmd")
		if err != nil {
			panic(err)
		}

		cmd := &exec.Cmd{
			Path: cmdpath,
			Args: []string{"/c", "start", "mesh.exe", config.Topic,fmt.Sprintf("%f",confignode.ConnectionProbability),fmt.Sprintf("%t",confignode.Gossip),fmt.Sprintf("%t",confignode.Subscribe),fmt.Sprintf("%t",confignode.Mode),fmt.Sprintf("%d",confignode.MsgInterval),fmt.Sprintf("%d",confignode.TestLength)},
		}
		cmd.Start()

		if err != nil {
			panic(err)
		}
	}

	fmt.Println("nodes are started")
}


func main() {


	exitch := make(chan bool,1)
	go monitorexit(exitch)


	http.HandleFunc("/signup",signup)
	http.HandleFunc("/done",isdone)
	http.HandleFunc("/results",handleresults)
	http.HandleFunc("/next",startnexttest)

	go http.ListenAndServe("localhost:8090",nil)

	fmt.Scanln()
	exitch <- true

}


func monitorexit(ch chan bool){
	for range ch{
		os.Exit(0)
	}
}


