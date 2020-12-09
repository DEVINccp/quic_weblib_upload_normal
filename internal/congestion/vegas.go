package congestion

const (
	vegasAlpha = 2
	vegasBeta = 4
	vegasGamma = 1
)

type Vegas struct {
	begSndNxt uint32
	begSndUna uint32
	begSndCwnd uint32
	doingVegasNow uint8
	cntRTT uint16
	minRTT uint32
	//minimum RTT in this connection
	baseRTT uint32
}

func vegasEnable() {

}

func vegasDisable() {

}

func vegasInit() {

}

func vegasState() {

}

func vegasCwndEvent() {

}

func vegasPktsAcked() {

}
