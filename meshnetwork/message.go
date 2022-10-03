package meshnetwork

type MeshMessage struct {
	Message   string
	SenderID  string
	Timestamp int64
}

func NewMeshMessage(msg string, senderid string, timestamp int64) MeshMessage {
	retval := MeshMessage{Message: msg, SenderID: senderid, Timestamp: timestamp}
	return retval
}