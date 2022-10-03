package main

import (
	"flag"
	"fmt"
	"mesh/configuration"
	"os/exec"
)

func main() {
	fnameflag := flag.String("f", "config.json", "passes the configuration file")
	flag.Parse()
	fname := *fnameflag

	config, err := configuration.NewMeshConfig(fname)
	if err != nil{panic(err)}

	fmt.Println("starting the nodes")
	for _,confignode := range config.Nodes{
		
		cmd := &exec.Cmd{
			Path:"cmd",
			Args: []string{"/c","start","wt","-d",".","./mesh.exe",config.Topic,fmt.Sprintf("%f",confignode.ConnectionProbability),fmt.Sprintf("%t",confignode.Gossip)},
		}
		err := cmd.Start()
		if err != nil{panic(err)}
	}

	fmt.Println("nodes are started")


}