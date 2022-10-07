package main

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	configuration "mesh/configuration"
	"mesh/hostrpc/rpc"
	"mesh/meshnetwork"
	"net/http"
	"os"
	"strconv"
	"time"
)


func main() {
	fmt.Print("Press Enter to exit...")
	rand.Seed(time.Now().UnixMilli())

	cli := make(chan string,1)

	ctx,cancel := context.WithCancel(context.Background())
	

	if len(os.Args) != 7{
		panic("uncompatible number of command line arguments")
	}
	topic := os.Args[1]
	probability,err := strconv.ParseFloat(os.Args[2],32)
	if err != nil{panic(err)}

	gossip, err := strconv.ParseBool(os.Args[3])
	if err != nil{panic(err)}

	sub, err := strconv.ParseBool(os.Args[4])
	if err != nil{panic(err)}

	mode, err := strconv.ParseBool(os.Args[5])
	if err != nil{panic(err)}

	interval, err := strconv.ParseInt(os.Args[6],10,32)
	if err != nil{panic(err)}

	confNode := configuration.ConfigNode{
		Topic: topic,
		ConnectionProbability: float32(probability),
		Gossip: gossip,
		Subscribe: sub,
		Mode:mode,
		MsgInterval: interval,
	}

	modestr := ""

	if mode{
		modestr = "unstructured sending"
	}else{
		modestr = "round robin sending"
	}

	fmt.Printf("\nTopic:%s",topic)	
	fmt.Printf("\nGossip:%t",gossip)	
	fmt.Printf("\nSubbed:%t",sub)	
	fmt.Printf("\nProbability:%f",probability)
	fmt.Printf("\nMode:%s",modestr)
	fmt.Printf("\nSending interval:%d",interval)


	go monitorcli(cli)

	// if ok, err := exists(meshnetwork.Tracesdir); ok && err == nil{
	// 	if e := os.RemoveAll(meshnetwork.Tracesdir); e != nil{
	// 		panic(e)
	// 	}
	// }
	
	// if err := os.MkdirAll(meshnetwork.Tracesdir,os.ModePerm);err != nil{
	// 	panic(err)
	// }


	node,err := meshnetwork.NewNode(cli,ctx,confNode)

	if err != nil{
		panic(err)
	}


	_, err = http.Post("http://localhost:8090/signup","application/text",bytes.NewBuffer([]byte(node.Getp2paddrs()[0].String())))
	if err != nil{
		panic(err)
	}
	c := rpc.NewControlService(node)
	
	rpc.Start(c)
	fmt.Scanln()
	cancel()
	close(cli)

	stats, err := node.GetStatsByTopicname(topic)
	if err == nil{
		fmt.Printf("\nTotal number of messages:%d",stats.TotalMsgCount)
		fmt.Printf("\nNumber of non-local messges:%d",stats.NonLocalMsgCount)
		fmt.Printf("\nAverage message passing latency:%d ms",stats.AverageDuration)
		fmt.Printf("\nMinimum message passing latency:%d ms",stats.MinDuration)
		fmt.Printf("\nMaximum message passing latency:%d ms\n",stats.MaxDuration)
	}

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


