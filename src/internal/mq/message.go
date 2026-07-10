package mq

type Msg struct {
	Topic   string
	Qos     byte
	Payload []byte
}
