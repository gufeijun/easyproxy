package easyproxy

import (
	"encoding/binary"
	"errors"
)

var ErrShortBuf = errors.New("Buf is too short")

type Packet struct {}

//均采取小端写
func NewPacket()*Packet{
	return &Packet{}
}

func (p *Packet)Write(buf []byte,nums ...uint32)error{
	if len(buf)/4<len(nums){
		return ErrShortBuf
	}
	for i:=0;i<len(nums);i++{
		binary.LittleEndian.PutUint32(buf[i*4:(i+1)*4],nums[i])
	}
	return nil
}

func (p *Packet)Read(buf []byte)uint32{
	num:=binary.LittleEndian.Uint32(buf)
	return num
}