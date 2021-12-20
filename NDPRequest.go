package main

type NDPType int

const (
	NDP_ADV NDPType = 0
	NDP_SOL NDPType = 1
)

type NDRequest struct {
	requestType NDPType
	//TODO use global unicast for router advertisements
	srcIP          []byte
	answeringForIP []byte
	mac            []byte
}
