package pndp

type ndpType int

const (
	ndp_ADV ndpType = 0
	ndp_SOL ndpType = 1
)

type ndpRequest struct {
	requestType      ndpType
	srcIP            []byte
	answeringForIP   []byte
	dstIP            []byte
	mac              []byte
	receivedIfaceMac []byte
	sourceIface      string
	rawPacket        []byte
}

type ndpQuestion struct {
	targetIP []byte
	askedBy  []byte
}
