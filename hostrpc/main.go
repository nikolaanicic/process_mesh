package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"mesh/configuration"
	rpc "mesh/hostrpc/rpc"
	mesh "mesh/meshnetwork"

	"github.com/libp2p/go-libp2p"
	gorpc "github.com/libp2p/go-libp2p-gorpc"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)



var peers []multiaddr.Multiaddr
var mx sync.Mutex = sync.Mutex{}

var signedup chan bool = make(chan bool,1)
var testsdonech chan bool = make(chan bool,1) 

var nodecount = 0

var results []mesh.Stats
var resmx sync.Mutex = sync.Mutex{}

type Monitor struct{
	IP string `json:"IP"`
}

type CoordinatorConfig struct{
	MonitorCount int32  `json:"MonitorCount"`
	Monitors []Monitor `json:"Monitors"`
	TestDir string `json:"TestDir"`
	ResultsDir string `json:"ResultsDir"`
}

type Result struct{
	Stats mesh.Stats `json:"Stats"`
	TestConfig configuration.CommonConfigFields `json:"Config"`
}


func main() {
	host := startclient()

	coordinatorConfig,err := loadCoordinatorConfig("coordconfig.json")
	if err != nil{panic(err)}
	fmt.Println("loaded the coordinators configuration")

	tests, _ := loadTestConfigurations(coordinatorConfig.TestDir)
	fmt.Println("loaded all the tests")
	fmt.Println("test count",len(tests))

	http.HandleFunc("/signup",signup)
	http.HandleFunc("/results",handleResult)

	go http.ListenAndServe("192.168.9.101:10000",nil)

	resdir := path.Join(coordinatorConfig.ResultsDir)
	createdirhierarchy(resdir)

	for _,testconfig := range tests{
		

		resmx.Lock()
		mx.Lock()

		results = []mesh.Stats{}

		peers = []multiaddr.Multiaddr{}
		nodecount = int( float32(testconfig.Nodecount) * testconfig.Subscribed)


		mx.Unlock()
		resmx.Unlock()
		fmt.Println("sending configurations")
		fmt.Println(coordinatorConfig.Monitors)

		for _, monitor := range coordinatorConfig.Monitors{
			sendconfig(monitor.IP,testconfig)
		}


		if !testconfig.Mode{
			ctx,cancel := context.WithDeadline(context.Background(),time.Now().Add(time.Second * time.Duration(testconfig.TestLength)))
			fmt.Println("waiting for the nodes to signup")
			<- signedup
			ticker := time.NewTicker(time.Millisecond * time.Duration(testconfig.MsgInterval))
			mx.Lock()
			fmt.Println("starting to manage the nodes")
			go manage(*ticker,peers,host,ctx,testconfig.Topic)
			go func(timer time.Timer,cancel context.CancelFunc){
				<-timer.C
				cancel()

			}(*time.NewTimer(time.Duration(testconfig.TestLength) * time.Second),cancel)
			mx.Unlock()
		}

		fmt.Println("waiting for the results")

		<- testsdonech
		
		resmx.Lock()
		agg := aggregateresults()
		resmx.Unlock()
		saveres(Result{Stats: agg,TestConfig: testconfig},path.Join(resdir,testconfig.TestName) + ".json")
	}

}



func signup(w http.ResponseWriter,req *http.Request){
	peeraddrbytes, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil{return}

	ma,err := multiaddr.NewMultiaddr(string(peeraddrbytes))
	if err != nil{return}

	mx.Lock()
	peers = append(peers, ma)
	fmt.Println(string(peeraddrbytes))
	if len(peers) == nodecount{
		signedup <- true
	}

	mx.Unlock()
}


func handleResult(w http.ResponseWriter,req *http.Request){
	resbytes, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	if err != nil{return}

	var result mesh.Stats
	if err := json.Unmarshal(resbytes,&result);err != nil{return}

	resmx.Lock()
	results = append(results, result)
	fmt.Println(results)
	if len(results) == nodecount{
		testsdonech <- true
	}
	resmx.Unlock()
}





func sendconfig(ip string,config configuration.CommonConfigFields){

	meshconfig,err := configuration.NewMeshConfigFromCommon(config)
	if err != nil{return}

	jsonbytes, err := json.Marshal(meshconfig)
	if err != nil{panic(err)}
	_,err = http.Post(ip + "/next","application/json",bytes.NewBuffer(jsonbytes))
	if err != nil{panic(err)}
}



func manage(ticker time.Ticker,peers []multiaddr.Multiaddr,h host.Host,ctx context.Context,topicname string){

	idx := 0
	plen := len(peers)
	for {
		select{
		case <-ticker.C:
			publish(h,peers[idx],topicname)
			idx = (idx + 1) % plen
		case <-ctx.Done():
			return
		}
	}
}



func createPeer(listenAddr string) host.Host {
	// Create a new libp2p host
	h, err := libp2p.New(libp2p.ListenAddrStrings(listenAddr),libp2p.NATPortMap())
	if err != nil {
		panic(err)
	}
	return h
}


func startclient() host.Host{
	client := createPeer("/ip4/0.0.0.0/tcp/0")
	fmt.Println("started a client on:",client.ID().Pretty())
	return client
}


func connect(client host.Host,to multiaddr.Multiaddr) *gorpc.Client{

	peerInfo, err := peer.AddrInfoFromP2pAddr(to)
	if err != nil{
		panic(err)
	}

	ctx := context.Background()
	err = client.Connect(ctx,*peerInfo)
	if err != nil{
		panic(err)
	}

	return gorpc.NewClient(client,rpc.ProtocolID)
}



func publish(me host.Host,to multiaddr.Multiaddr,topicname string){
	client := connect(me,to)
	call(client,"ControlService","Publish",rpc.Args{Topicname: topicname,Message: ""},to)
}






func call(client *gorpc.Client,svcname string,method string,args rpc.Args,to multiaddr.Multiaddr){

	peerinfo,err := peer.AddrInfoFromP2pAddr(to)
	if err != nil{
		return 
	}

	reply := &rpc.Reply{}
	err = client.Call(peerinfo.ID,svcname,method,args,reply)
	if err != nil{
		fmt.Println(err.Error())
		return 
	}
	fmt.Println(reply)
}



func loadTestConfigurations(dir string) ([]configuration.CommonConfigFields,error){
	files, err := os.ReadDir(dir)
	if err != nil{return nil,err}

	var configs []configuration.CommonConfigFields

	for _,file := range files{
		config, err := loadTestConfiguration(path.Join(dir,file.Name()))
		if err != nil{panic(err)}
		configs = append(configs, config)
	}

	return configs,nil
}


func loadTestConfiguration(filename string) (configuration.CommonConfigFields, error){
	filebytes,err := os.ReadFile(filename)

	if err != nil{return configuration.CommonConfigFields{}, err}

	var config configuration.CommonConfigFields
	if err := json.Unmarshal(filebytes,&config); err != nil{return configuration.CommonConfigFields{},err}
	return config,nil
}


func loadCoordinatorConfig(filename string) (CoordinatorConfig,error){
	filebytes,err := os.ReadFile(filename)
	if err != nil{return CoordinatorConfig{}, err}

	var coordconfig CoordinatorConfig
	if err := json.Unmarshal(filebytes,&coordconfig); err != nil{return coordconfig,err}

	return coordconfig,nil
}


func aggregateresults() mesh.Stats{

	res := mesh.Stats{AverageDuration: 0}
	for _,r := range results{
		res.AverageDuration += r.AverageDuration
		res.NonLocalMsgCount += r.NonLocalMsgCount
		res.TotalMsgCount += r.TotalMsgCount
	}
	
	res.AverageDuration = res.AverageDuration / int64(len(results))
	return res
}

func saveres(res Result,resfile string){

	statbytes, err := json.Marshal(res)
	if err != nil{panic(err)}

	if err := ioutil.WriteFile(resfile,statbytes,os.ModePerm);err != nil{panic(err)}
}


func createdirhierarchy(path string){
		if err := os.MkdirAll(path,os.ModePerm);err != nil{
		panic(err)
	}
}

