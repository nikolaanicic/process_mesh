package meshnetwork

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// size of the buffer in which the messages are going to be received
const TopicBufferSize = 128

type MeshTopic struct{
	Ctx context.Context
	PubSub *pubsub.PubSub
	Sub *pubsub.Subscription
	Topic *pubsub.Topic
	Topicname string
	Self peer.ID
	Messages chan *MeshMessage
	CLI chan string
	avgduration int64
	msgcnt int64
}


func (t *MeshTopic) IsSubscribed() bool{
	return t.Sub != nil
}

func (t *MeshTopic) Subscribe() error{
	sub, err := t.Topic.Subscribe()
	if err != nil{return err}

	t.Sub = sub
	go t.topicmessagehandler()
	return nil
}


func (t *MeshTopic) GetAverageDuration() int64{

	if t.msgcnt != 0 {
		return t.avgduration / t.msgcnt
	}

	return -1
}

func (t *MeshTopic) Publish(message string) error{

	m := &MeshMessage{
		Message: message,
		SenderID: t.Self.ShortString(),
		Timestamp: time.Now().UTC().UnixMilli(),
	}

	msg, err := json.Marshal(m)
	if err != nil{
		return err
	}

	return t.Topic.Publish(t.Ctx,msg)
}

func (t *MeshTopic) Unsubscribe(){
	if t.IsSubscribed(){
		t.Sub.Cancel()
	}
}


func (t *MeshTopic) print(msg string){
	t.CLI <- msg
}

func (t *MeshTopic) formatnewmesssage(msg *MeshMessage,duration int64) string{

	return fmt.Sprintf("\n<%s>\t<%s>\t%s\t\t<%s>\t<%d ms>",t.Topicname,t.Self.Pretty(),msg.Message,msg.SenderID,duration)
}




func (t *MeshTopic) tryclosemessages(){
	_,ok := <-t.Messages
	if ok{close(t.Messages)}
}

func (t *MeshTopic) topicmessagehandler(){
	t.avgduration = 0
	t.msgcnt = 0

	for{
		msg, err := t.Sub.Next(t.Ctx)
		if err != nil{
			t.tryclosemessages()
			return
		}

		if msg.ReceivedFrom == t.Self{continue}
		

		m := new(MeshMessage)
		err = json.Unmarshal(msg.Data,m)
		if err != nil{continue}
		duration := time.Now().UTC().UnixMilli() - m.Timestamp

		

		t.msgcnt++
		if duration < 0{duration = -duration}
		t.avgduration += duration

		
		//t.print(t.formatnewmesssage(m,duration))
	}
}


func JoinTopic(ctx context.Context, ps *pubsub.PubSub,topicname string,self peer.ID,cli chan string)(*MeshTopic, error){
	topic, err := ps.Join(topicname)
	if err != nil{return nil, err}

	return &MeshTopic{
		Ctx: ctx,
		PubSub: ps,
		Sub: nil,
		Self:self,
		Topicname: topicname,
		Messages: make(chan *MeshMessage,TopicBufferSize),
		CLI:cli,
		Topic: topic,
	}, nil
}