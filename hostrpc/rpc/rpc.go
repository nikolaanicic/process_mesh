package rpc

import (
	"context"
	"fmt"
	"time"

	mesh "mesh/meshnetwork"

	gorpc "github.com/libp2p/go-libp2p-gorpc"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var ProtocolID = protocol.ID("/p2p/rpc/nodecontrol")


type RpcCodes int

const (
	Success RpcCodes = iota
	Failure
)


type Args struct {
	Topicname string
	Message   string
}

type ControlService struct {
	Node *mesh.MeshNode
}

type Reply struct {
	Code RpcCodes
	Message  string
}


func formatmesssage(n *mesh.MeshNode,msg string) string{
	return fmt.Sprintf("<%s>:%s",n.Host.ID().Pretty(),msg)
}


func (c *ControlService) Publish(ctx context.Context, args Args, reply *Reply) error {

	if c.Node.TryPublishOnTopic(args.Topicname){
		reply.Code = Success
		reply.Message = formatmesssage(c.Node,"OK")
	}else{
		reply.Code = Failure
		reply.Message = formatmesssage(c.Node,"failed to publish")
	}

	return fmt.Errorf(reply.Message)
}




func Start(c *ControlService) {
	rpcHost := gorpc.NewServer(c.Node.Host, ProtocolID)
	err := rpcHost.Register(c)

	if err != nil {
		c.Node.CLI <- err.Error()
		return
	}

	go func() {
		for {
			time.Sleep(time.Millisecond * 5)
		}
	}()

	go c.Node.Run()
}

func Print(c *ControlService){
	fmt.Println(c.Node)
}


func NewControlService(n *mesh.MeshNode) *ControlService{
	return &ControlService{Node:n}
}