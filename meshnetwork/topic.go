package meshnetwork

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// size of the buffer in which the messages are going to be received
const TopicBufferSize = 128


type Stats struct{
	TotalMsgCount int64
	NonLocalMsgCount int64
	AverageDuration int64
	MinDuration int64
	MaxDuration int64
}

type TestResults struct{
	Results [] Stats `json:"Results"`
}




type MeshTopic struct{
	Ctx context.Context
	PubSub *pubsub.PubSub
	Sub *pubsub.Subscription
	Topic *pubsub.Topic
	Topicname string
	Self peer.ID
	Messages chan *MeshMessage
	CLI chan string
	topicStats Stats
}

func newStats() Stats{
	return Stats{
		TotalMsgCount: 0,
		NonLocalMsgCount: 0,
		AverageDuration: 0,
		MinDuration: math.MaxInt64,
		MaxDuration: math.MinInt64,
	}
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


func (t *MeshTopic) GetStats() Stats{
	if t.topicStats.NonLocalMsgCount > 0{
		t.topicStats.AverageDuration = t.topicStats.AverageDuration/t.topicStats.NonLocalMsgCount
	}

	return t.topicStats
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


func (t *MeshTopic) getnextmessage(ch chan *pubsub.Message) {

	for{
		msg,err := t.Sub.Next(t.Ctx)
		if err != nil{
			t.tryclosemessages()
			return
		}
		ch <- msg
	}

}

func (t *MeshTopic) topicmessagehandler(){

	msgchan := make(chan *pubsub.Message,1)
	go t.getnextmessage(msgchan)

	for{
		select{
		case msg := <-msgchan:
			t.topicStats.TotalMsgCount++

			if msg.ReceivedFrom == t.Self || msg.Local {
				continue
			}
	
			m := new(MeshMessage)
			if err := json.Unmarshal(msg.Data,m);err != nil{continue}
	
			duration := time.Now().UTC().UnixMilli() - m.Timestamp
	
			if duration < 0{duration = -duration}
	
			t.topicStats.AverageDuration += duration
			t.topicStats.NonLocalMsgCount++
	
			if duration > 0 && duration < t.topicStats.MinDuration{
				t.topicStats.MinDuration = duration
			}
	
			if duration > 0 && duration > t.topicStats.MaxDuration{
				t.topicStats.MaxDuration = duration
			}
			//t.print(t.formatnewmesssage(m,duration))
		case <-t.Ctx.Done():
			return
		}
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
		topicStats: newStats(),
	}, nil
}