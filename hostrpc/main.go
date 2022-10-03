package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	rpc "mesh/hostrpc/rpc"

	"github.com/libp2p/go-libp2p"
	gorpc "github.com/libp2p/go-libp2p-gorpc"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)


var done chan bool

type Method int

const (
	Publish Method = iota
	Exit
)









func main() {
	host := startclient()

	ticker := time.NewTicker(time.Millisecond * 10)
	peers,err := loadFromFile("peers.txt")
	if err != nil{panic(err)}

	done = make(chan bool, 1)
	fmt.Println("loaded the peers.press enter to start the round robin")
	fmt.Scanln()
	go manage(*ticker,peers,host)
	fmt.Println("Press Enter to exit...")
	
	fmt.Scanln()
	done <- true
	time.Sleep(time.Millisecond)

	close(done)

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


func exit(h host.Host){
	err := h.Close()
	if err !=nil{panic(err)}
	os.Exit(0)
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



func menu() Method {
	retval := Exit
	loop := true

	for loop{
		fmt.Println("1.Publish")
		fmt.Println("2.Close the client")

		var input string
		fmt.Scanln(&input)
		i,err := strconv.ParseInt(input,10,64)
		if err == nil && i-1 >= int64(Publish) && i-1 <= int64(Exit){
			retval = Method(i)
			loop = false
		}else{loop = true}
	}

	return Method(retval-1)
}



