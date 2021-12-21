package main

type NDPType int

const (
	NDP_ADV NDPType = 0
	NDP_SOL NDPType = 1
)

type NDRequest struct {
	requestType      NDPType
	srcIP            []byte
	answeringForIP   []byte
	dstIP            []byte
	mac              []byte
	receivedIfaceMac []byte
	sourceIface      string
}
