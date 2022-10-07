package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
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


var done chan bool

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


func main() {
	host := startclient()

	coordinatorConfig,err := loadCoordinatorConfig("coordconfig.json")
	if err != nil{panic(err)}

	tests, _ := loadTestConfigurations(coordinatorConfig.TestDir)


	http.HandleFunc("/signup",signup)
	go http.ListenAndServe(":10000",nil)


	for _,testconfig := range tests{
		

		resmx.Lock()
		mx.Lock()

		results := []mesh.Stats{}

		peers := []multiaddr.Multiaddr{}
		nodecount = testconfig.Nodecount

		mx.Unlock()
		resmx.Unlock()

		for _, monitor := range coordinatorConfig.Monitors{
			sendconfig(monitor.IP,testconfig)
		}

		<- signedup

		if !testconfig.Mode{
			ticker := time.NewTicker(time.Millisecond * time.Duration(testconfig.MsgInterval))
			mx.Lock()
			go manage(*ticker,peers,host)
			mx.Unlock()
		}

		<- testsdonech
		resdir := path.Join(coordinatorConfig.ResultsDir,testconfig.TestName,".json")
		createdirhierarchy(resdir)
		
		resmx.Lock()
		agg := aggregateresults(results)
		resmx.Unlock()
		saveres(agg,resdir)
	}


	close(done)

}



func signup(w http.ResponseWriter,req *http.Request){
	defer req.Body.Close()
	peeraddrbytes, err := ioutil.ReadAll(req.Body)

	if err != nil{return}

	ma,err := multiaddr.NewMultiaddr(string(peeraddrbytes))
	if err != nil{return}

	mx.Lock()
	peers = append(peers, ma)
	
	if len(peers) == nodecount{
		signedup <- true
	}

	mx.Unlock()
}



func sendconfig(ip string,configuration configuration.CommonConfigFields){
	jsonbytes, err := json.Marshal(configuration)
	if err != nil{panic(err)}
	http.Post(ip + "/next","application/json",bytes.NewBuffer(jsonbytes))
}


func manage(ticker time.Ticker,peers []multiaddr.Multiaddr,h host.Host){

	idx := 0
	plen := len(peers)
	for {
		select{
		case <-ticker.C:
			publish(h,peers[idx])
			idx = (idx + 1) % plen
		case <-done:
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



func loadFromFile(filename string) ([]multiaddr.Multiaddr,error){

	var retval []multiaddr.Multiaddr
	file,err := os.Open(filename)
	if err != nil{return nil,err}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan(){
		straddr := scanner.Text()
		m,err := multiaddr.NewMultiaddr(straddr)
		if err !=nil{panic(err)}
		retval = append(retval, m)
	
	}

	return retval,nil
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



func publish(me host.Host,to multiaddr.Multiaddr){
	client := connect(me,to)
	call(client,"ControlService","Publish",rpc.Args{Topicname: "lotr",Message: ""},to)
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


func randstring() string{
	charset := "abcdefghijklmnopqrstuvx"
	l := 10
	str := ""

	for i:=0;i<l;i++{
		str += string(charset[rand.Intn(len(charset))])
	}

	return str

}

func aggregateresults(results []mesh.Stats) mesh.Stats{

	res := mesh.Stats{}
	for _,r := range results{
		res.AverageDuration += r.AverageDuration
	}
	res.AverageDuration = res.AverageDuration / int64(len(results))

	return res
}

func saveres(res mesh.Stats,resfile string){

	statbytes, err := json.Marshal(res)
	if err != nil{panic(err)}

	if err := ioutil.WriteFile(resfile,statbytes,os.ModePerm);err != nil{panic(err)}
}


func exists(path string) (bool,error){
	_, err := os.Stat(path)
	if err == nil{return true,nil}
	if os.IsNotExist(err){return false,nil}
	return false, err
}

func createdirhierarchy(path string){
		if err := os.MkdirAll(path,os.ModePerm);err != nil{
		panic(err)
	}
}

