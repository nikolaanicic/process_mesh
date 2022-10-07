package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"mesh/configuration"
	"net/http"
	"os"
	"os/exec"
	"sync"
)


var mx sync.Mutex = sync.Mutex{}
var ch chan string = make(chan string,1)

func signup(w http.ResponseWriter,req *http.Request){

	host,err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	if err == nil{
		mx.Lock()
		fmt.Println(string(host))

		mx.Unlock()
	}
	fmt.Fprintf(w,"OK")
	
}



func startnodes(){
	fnameflag := flag.String("f", "config.json", "passes the configuration file")
	flag.Parse()
	fname := *fnameflag

	config, err := configuration.NewMeshConfig(fname)
	if err != nil{panic(err)}


	fmt.Println("starting the nodes")
	for _,confignode := range config.Nodes{
		
		cmdpath, err := exec.LookPath("cmd")
		if err != nil {
			panic(err)
		}

		cmd := &exec.Cmd{
			Path: cmdpath,
			Args: []string{"/c", "start", "mesh.exe", config.Topic,fmt.Sprintf("%f",confignode.ConnectionProbability),fmt.Sprintf("%t",confignode.Gossip),fmt.Sprintf("%t",confignode.Subscribe),fmt.Sprintf("%t",confignode.Mode),fmt.Sprintf("%d",confignode.MsgInterval)},
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
	go monitorcli()

	go startnodes()

	http.HandleFunc("/signup",signup)
	go http.ListenAndServe("localhost:8090",nil)


	fmt.Scanln()
	exitch <- true

}


func monitorcli(){
	for range ch{
		fmt.Println(ch)
	}
}

func monitorexit(ch chan bool){
	for range ch{
		os.Exit(0)
	}
}


