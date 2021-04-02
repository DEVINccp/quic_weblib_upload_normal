package networkMeasure

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

const (
	MAX_PG = 2000
)

// 封装 icmp 报头
type ICMP struct {
	Type        uint8
	Code        uint8
	Checksum    uint16
	Identifier  uint16
	SequenceNum uint16
}

var (
	originBytes []byte
)

func init() {
	originBytes = make([]byte, MAX_PG)
}

func CheckSum(data []byte) (rt uint16) {
	var (
		sum    uint32
		length int = len(data)
		index  int
	)
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		index += 2
		length -= 2
	}
	if length > 0 {
		sum += uint32(data[index]) << 8
	}
	rt = uint16(sum) + uint16(sum>>16)

	return ^rt
}

func Ping(domain string, PS, Count int) (float64,float64){
	var (
		icmp                   ICMP
		laddr                  = net.IPAddr{IP: net.ParseIP("0.0.0.0")} // 得到本机的IP地址结构
		raddr, _               = net.ResolveIPAddr("ip", domain)        // 解析域名得到 IP 地址结构
		maxLan, minLan, avgLan float64
	)

	// 返回一个 ip socket
	conn, err := net.DialIP("ip4:icmp", &laddr, raddr)

	if err != nil {
		fmt.Println(err.Error())
		return 0.0,0
	}

	defer conn.Close()

	// 初始化 icmp 报文
	icmp = ICMP{8, 0, 0, 0, 0}

	var buffer bytes.Buffer
	binary.Write(&buffer, binary.BigEndian, icmp)
	//fmt.Println(buffer.Bytes())
	binary.Write(&buffer, binary.BigEndian, originBytes[0:PS])
	b := buffer.Bytes()
	binary.BigEndian.PutUint16(b[2:], CheckSum(b))

	//fmt.Println(b)
	fmt.Printf("\n正在 Ping %s 具有 %d(%d) 字节的数据:\n", raddr.String(), PS, PS+28)
	recv := make([]byte, 1024)
	var retList []float64

	dropPack := 0.0 /*统计丢包的次数，用于计算丢包率*/
	maxLan = 3000.0
	minLan = 0.0
	avgLan = 0.0

	for i := Count; i > 0; i-- {
		/*
			向目标地址发送二进制报文包
			如果发送失败就丢包 ++
		*/
		if _, err := conn.Write(buffer.Bytes()); err != nil {
			dropPack++
			time.Sleep(time.Second)
			continue
		}
		// 否则记录当前得时间
		tStart := time.Now()
		conn.SetReadDeadline(time.Now().Add(time.Second * 3))
		len, err := conn.Read(recv)
		/*
			查目标地址是否返回失败
			如果返回失败则丢包 ++
		*/
		if err != nil {
			dropPack++
			time.Sleep(time.Second)
			continue
		}
		tEnd := time.Now()
		dur := float64(tEnd.Sub(tStart).Nanoseconds()) / 1e6
		retList = append(retList, dur)
		if dur < maxLan {
			maxLan = dur
		}
		if dur > minLan {
			minLan = dur
		}
		fmt.Printf("来自 %s 的回复: 大小 = %d byte 时间 = %.3fms\n", raddr.String(), len ,dur)
		time.Sleep(time.Second)
	}
	lossRate := dropPack/float64(Count)*100
	fmt.Printf("丢包率: %.2f%%\n", lossRate)
	if len(retList) == 0 {
		avgLan = 3000.0
	} else {
		sum := 0.0
		for _, n := range retList {
			sum += n
		}
		avgLan = sum / float64(len(retList))
	}
	fmt.Printf("rtt 最短 = %.3fms 平均 = %.3fms 最长 = %.3fms\n", minLan, avgLan, maxLan)
	return lossRate, avgLan
}

func NetworkSituation(IP string) (float64,float64){
	lossRate, avgDelay:= Ping(IP, 48, 5)
	return lossRate,avgDelay
}