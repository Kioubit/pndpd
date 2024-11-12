package pndp

type ndpType int

const (
	ndpAdv ndpType = 0
	ndpSol ndpType = 1
)

type ndpRequest struct {
	requestType    ndpType
	srcIP          []byte
	answeringForIP []byte
	dstIP          []byte
	sourceIface    string
	payload        []byte
}

type ndpQuestion struct {
	targetIP []byte
	askedBy  []byte
}
