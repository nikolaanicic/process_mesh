package main

import (
	"context"
	"fmt"
	"math/rand"
	configuration "mesh/configuration"
	"mesh/hostrpc/rpc"
	"mesh/meshnetwork"
	"os"
	"strconv"
	"time"
)

func main() {
	fmt.Print("Press Enter to exit...")
	rand.Seed(time.Now().UnixMilli())

	cli := make(chan string,1)

	ctx := context.Background()
	

	if len(os.Args) != 4{
		panic("uncompatible number of command line arguments")
	}

	topic := os.Args[1]
	probability,err := strconv.ParseFloat(os.Args[2],32)
	if err != nil{panic(err)}

	gossip, err := strconv.ParseBool(os.Args[3])
	if err != nil{panic(err)}

	confNode := configuration.ConfigNode{
		Topic: topic,
		ConnectionProbability: float32(probability),
		Gossip: gossip,
	}


	go monitorcli(cli)

	if ok, err := exists(meshnetwork.Tracesdir); ok && err == nil{
		if e := os.RemoveAll(meshnetwork.Tracesdir); e != nil{
			panic(e)
		}
	}
	
	if err := os.MkdirAll(meshnetwork.Tracesdir,os.ModePerm);err != nil{
		panic(err)
	}


	node,err := meshnetwork.NewNode(cli,ctx,confNode)

	if err != nil{
		panic(err)
	}

	
	c := rpc.NewControlService(node)
	rpc.Start(c)
	fmt.Scanln()

	avg, err := node.GetAverageMsgTimeByTopicname(topic)
	if err != nil{
		fmt.Printf("\nAverage time:%d ms",avg)
	}
	close(cli)

	fmt.Println("Press Enter to exit...")
	fmt.Scanln()
}


func monitorcli(cli chan string){
	for m := range cli{
		fmt.Print(m)
	}
}

func exists(path string) (bool,error){
	_, err := os.Stat(path)
	if err == nil{return true,nil}
	if os.IsNotExist(err){return false,nil}
	return false, err
}


