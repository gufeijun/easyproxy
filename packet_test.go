package easyproxy

import (
	"testing"
)

func TestPacket_Write(t *testing.T) {
	packet:=NewPacket()
	type Test struct {
		input []uint32
		want []uint32
	}
	tests:=[]Test{
		{[]uint32{0,1,2,3,4},[]uint32{0,1,2,3,4}},
		{[]uint32{5},[]uint32{5}},
	}
	for _,test:=range tests{
		buf:=make([]byte,len(test.input)*4)
		if err:=packet.Write(buf,test.input...);err!=nil{
			t.Fatal(err)
		}
		for i:=0;i<len(test.input);i++{
			got:=packet.Read(buf[i*4:(i+1)*4])
			if got!=test.want[i]{
				t.Errorf("want %d,but got %d",test.want[i],got)
			}
		}
	}
	for _,test:=range tests{
		buf:=make([]byte,len(test.input)*4-1)
		if err:=packet.Write(buf,test.input...);err!=ErrShortBuf{
			t.Errorf("want err:%v , got err:%v",ErrShortBuf,err)
		}
	}
}