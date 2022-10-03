package meshnetwork

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	configuration "mesh/configuration"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
)


const DiscoveryInterval = time.Hour
const DiscoveryServiceTag = "mesh_v1"
const Tracesdir = "traces/"

type MeshNode struct {
	topics 			map[string]*MeshTopic
	Host            host.Host
	ctx 			context.Context
	CLI				chan string
	PS				*pubsub.PubSub
	P2paddr			multiaddr.Multiaddr
	cfg				configuration.ConfigNode
}


func NewNode(cli chan string,ctx context.Context,cfg configuration.ConfigNode)(*MeshNode,error){
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"), libp2p.NATPortMap(),libp2p.WithDialTimeout(time.Minute * 1))

	if err != nil{return nil, err}

	return &MeshNode{
		topics: make(map[string]*MeshTopic),
		Host:h,
		ctx: ctx,
		CLI:cli,
		PS:nil,
		cfg: cfg,
	},nil
}


func (n *MeshNode) randmessage() string{
	charset := "abcdefghijklmnopqrstuvwxyz"
	msg := ""
	for len(msg) < 250{
		msg += string(charset[rand.Intn(len(charset))])
	}

	return msg
}



func (n *MeshNode) TryPublishOnTopic(topicname string) bool{
	if topic, ok := n.topics[topicname];ok{
		if err := topic.Publish(n.randmessage()); err != nil{
			n.print(n.formatmesssage(err.Error()))
			return false
		}else{
			n.print(n.formatmesssage(fmt.Sprintf("published on %s",topicname)))
			return true
		}
	}
	return false
}


func (n *MeshNode) print(msg string){
	n.CLI <- msg
}


func (n *MeshNode) Jointopicname(t string) bool{

	if _,ok := n.topics[t];ok{
		return true
	}

	topic, err := JoinTopic(n.ctx,n.PS,t,n.Host.ID(),n.CLI)
	if err != nil{
		// n.print(n.formatmesssage(fmt.Sprintf("failed to join %s\terr:%s",t,err.Error())))
		return false
		
	}else{
		n.topics[t] = topic
		// n.print(n.formatmesssage(fmt.Sprintf("joined %s topic",t)))
		
		return true
	}
}


func (n *MeshNode) formatmesssage(msg string) string{

	return fmt.Sprintf("\n<%s>:%s",n.P2paddr,msg)
}


func (n *MeshNode) Getp2paddrs() []multiaddr.Multiaddr{
	var retval []multiaddr.Multiaddr
	for _, a := range n.Host.Addrs(){
		p2paddr, err := multiaddr.NewMultiaddr("/ipfs/" + n.Host.ID().Pretty())
		if err == nil{
			addr := a.Encapsulate(p2paddr)
			retval = append(retval,addr)
		}
	}

	return retval
}

func (n *MeshNode) PrintP2PAddrs(){
	retval := "\n" + strings.Repeat("-",20)
	for _, a := range n.Getp2paddrs(){
		retval += fmt.Sprintf("\n%s",a)
	}
	retval += "\n" + strings.Repeat("-",20)
	n.print(retval)
}

func (n *MeshNode) subtopic(topic *MeshTopic) bool{
	if !topic.IsSubscribed(){
		err := topic.Subscribe()
		// if err != nil{
		// 	n.print(n.formatmesssage(fmt.Sprintf("failed to subscribe to %s",err.Error())))
		// }else{
		// 	n.print(n.formatmesssage(fmt.Sprintf("subscribed %s topic",topic.Topicname)))
		// }
		return err != nil
	}
	return true
}

func (n *MeshNode) GetAverageMsgTimeByTopicname(topicname string) (int64,error){
	if topic,ok := n.topics[topicname];ok{
		return topic.GetAverageDuration(),nil
	}
	return -1,fmt.Errorf("invalid topicname")
}


func (n *MeshNode) EnterTopicname(topicname string){
	if n.Jointopicname(topicname){
		n.Subtopicname(topicname)
	}
}


func (n *MeshNode) Subtopicname(topicname string) bool{
	if topic, ok := n.topics[topicname];ok{
		return n.subtopic(topic)
	}
		
	return false
}



func (n *MeshNode) HandlePeerFound(pi peer.AddrInfo) {
		if rand.Float32() <= n.cfg.ConnectionProbability{
			err := n.Host.Connect(n.ctx, pi)
			if err != nil {
				n.print(n.formatmesssage(err.Error()))
			}
		}




}

func (n *MeshNode) setupLocalDiscovery() error {
	s := mdns.NewMdnsService(n.Host, DiscoveryServiceTag, n)
	return s.Start()
}

func (n *MeshNode) Shutdown() error{
	for _,t := range n.topics{
		t.Unsubscribe()
	}

	return n.Host.Close()
}


func (n *MeshNode) initrouter() error{
	
	var ps *pubsub.PubSub
	var e error

	if n.cfg.Gossip{
		ps, e = pubsub.NewGossipSub(n.ctx,n.Host)
	}else{
		ps,e = pubsub.NewFloodSub(n.ctx,n.Host)
	}

	if e != nil{return e}

	if err := n.setupLocalDiscovery(); err != nil{return err}
	n.PS = ps
	return nil
}


func (n *MeshNode) Run(){
	if err := n.initrouter(); err != nil{
		// n.print(n.formatmesssage(fmt.Sprintf("failed to init gossipsub router\terr:%s",err.Error())))
		return
	}

	n.P2paddr = n.Getp2paddrs()[0]

	n.CLI <- fmt.Sprintf("\n%s",n.P2paddr)
	n.EnterTopicname(n.cfg.Topic)


}