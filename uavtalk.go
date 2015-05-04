package main

import (
	"fmt"
	"log"

	"github.com/GeertJohan/go.hid"
)

// TODO: refactor for better value reading (encoding/binary)

const VER_MASK = 0x20

type UAVTalkObject struct {
	cmd        uint8
	length     uint16
	objectId   uint32
	instanceId uint16
	data       []byte
}

func byteArrayToInt32(b []byte) uint32 {
	if len(b) != 4 {
		panic("byteArrayToInt32 requires at least 4 bytes")
	}

	return (uint32(b[3]) << 24) | (uint32(b[2]) << 16) | (uint32(b[1]) << 8) | (uint32(b[0]))
}

func byteArrayToInt16(b []byte) uint16 {
	if len(b) != 2 {
		panic("byteArrayToInt16 requires at least 2 bytes")
	}

	return (uint16(b[1]) << 8) | (uint16(b[0]))
}

func packetComplete(packet []byte) (bool, int, int) {
	var offset int = -1
	for i := 0; i < len(packet); i++ {
		if packet[i] == 0x3c {
			offset = i
			break
		}
	}

	if offset < 0 {
		return false, 0, 0
	}

	frame := packet[offset:]

	if len(frame) < 11 {
		return false, 0, 0
	}

	length := byteArrayToInt16(frame[2:4])

	if int(length)+1 > len(frame) {
		return false, 0, 0
	}

	cks := frame[length]

	if cks != computeCrc8(0, packet[0:length]) {
		return false, 0, 0
	}

	return true, offset, offset + int(length) + 1
}

func newUAVTalkObject(packet []byte) (*UAVTalkObject, error) {
	uavTalkObject := &UAVTalkObject{}

	uavTalkObject.cmd = packet[1] ^ VER_MASK
	uavTalkObject.length = byteArrayToInt16(packet[2:4])
	uavTalkObject.objectId = byteArrayToInt32(packet[4:8])
	uavTalkObject.instanceId = byteArrayToInt16(packet[8:10])

	uavTalkObject.data = make([]byte, uavTalkObject.length-10)
	copy(uavTalkObject.data, packet[10:len(packet)-1])

	return uavTalkObject, nil
}

func printHex(buffer []byte, n int) {
	fmt.Println("start packet:")
	for i := 0; i < n; i++ {
		if i > 0 {
			fmt.Print(":")
		}
		fmt.Printf("%.02x", buffer[i])
	}
	fmt.Println("\nend packet")
}

func startHID(stopChan chan bool, uavChan chan *UAVTalkObject) {
	cc, err := hid.Open(0x20a0, 0x415b, "")
	if err != nil {
		log.Fatal(err)
	}
	defer cc.Close()

	buffer := make([]byte, 64)
	packet := make([]byte, 0, 4096)
	for {
		select {
		case _ = <-stopChan:
			return
		default:
		}

		n, err := cc.Read(buffer)
		if err != nil {
			panic(err)
		}

		packet = append(packet[len(packet):], buffer[2:n]...)

		for {
			ok, from, to := packetComplete(packet)
			if ok != true {
				break
			}
			//printHex(packet[from:to], to-from)
			if uavTalkObject, err := newUAVTalkObject(packet[from:to]); err == nil {
				uavChan <- uavTalkObject
			} else {
				fmt.Println(err)
			}
			copy(packet, packet[from:])
			packet = packet[0 : len(packet)-to]
		}
	}
}

var startStopControlChan = make(chan bool)

func openUAVChan() {
	startStopControlChan <- true
}

func closeUAVChan() {
	startStopControlChan <- false
}

func startUAVTalk(uavChan chan *UAVTalkObject) {
	go func() {
		stopChan := make(chan bool)
		started := false
		for startStop := range startStopControlChan {
			if startStop {
				if started == false {
					go startHID(stopChan, uavChan)
				}
				started = true
			} else {
				if started {
					stopChan <- true
					started = false
				}
			}
		}
	}()
}
