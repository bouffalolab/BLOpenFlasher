package utils

import(
    "fmt"
//    "os"
    "io/ioutil"
    "bytes"
    "encoding/binary"
	"encoding/hex"
    "hash/crc32"
    "github.com/pelletier/go-toml"
)

type partition struct{
    pt_table struct{
        address0 uint
        address1 uint
    }
    pt_entry []struct{
        type_ uint
        name string
        device uint
        address0 uint
        size0 uint
        address1 uint
        sieze1 uint
        len uint
    }
}

type GenPartition struct{
    IfName string
    OfName string
    BinAddress map[uint] string
}

func (this *GenPartition)CreatePartitionBin() bool{
    fmt.Println(this.IfName)
    fmt.Println(this.OfName)
    if this.BinAddress == nil {
        this.BinAddress = make(map[uint]string, 0)
    }

    tree, err := toml.LoadFile(this.IfName)
    if err != nil {
        fmt.Println(err)
        return false
    }
/*    
    tp := tree.Get("pt_table.address0")
    if tp == nil {
        return false
    }
    taddr0 := tp.(int64)
    tp = tree.Get("pt_table.address1")
    if tp == nil {
        return false
    }
    taddr1 := tp.(int64)
    tp = tree.Get("pt_table.bin0")
    if tp == nil {
        return false
    }
    tbin0 := tp.(string)
    tp = tree.Get("pt_table.bin1")
    if tp == nil {
        return false
    }
    tbin1 := tp.(string)
    fmt.Printf("%v %v %v %v\r\n", taddr0, taddr1, tbin0, tbin1)
    this.BinAddress[uint(taddr0)] = tbin0
    this.BinAddress[uint(taddr1)] = tbin1
*/
    t := tree.Get("pt_entry").([]*toml.Tree)
    outBytes := make([]byte, 16 + len(t) * 36 + 4)
    outBytes[0] = 0x42
    outBytes[1] = 0x46
    outBytes[2] = 0x50
    outBytes[3] = 0x54
    copy(outBytes[6:], intToBytes(len(t)&0xffff)[0:])
    copy(outBytes[12:16], intToBytes(int(crc32.ChecksumIEEE(outBytes[0:12])))[0:]) 

    for i:=0; i<len(t); i++ {
        vtype := t[i].Get("type").(int64)
        //fmt.Println(vtype)
        copy(outBytes[16+36*i+0:], intToBytes(int(vtype)))
        vname := t[i].Get("name").(string)
        //fmt.Println(vname)
        if len(vname)>8 {
            return false
        }
        copy(outBytes[16+36*i+3:], []byte(vname))
        //vdevice := t[i].Get("device").(int64)
        //fmt.Println(vdevice)
        vaddress0 := t[i].Get("address0").(int64)
        fmt.Printf("%x\r\n", vaddress0)
        copy(outBytes[16+36*i+12:], intToBytes(int(vaddress0)))
        vaddress1 := t[i].Get("address1").(int64)
        fmt.Printf("%x\r\n", vaddress1)
        copy(outBytes[16+36*i+16:], intToBytes(int(vaddress1)))
        vsize0 := t[i].Get("size0").(int64)
        fmt.Printf("%x\r\n", int(vsize0))
        copy(outBytes[16+36*i+20:], intToBytes(int(vsize0)))
        vsize1 := t[i].Get("size1").(int64)
        fmt.Printf("%x\r\n", int(vsize1))
        copy(outBytes[16+36*i+24:], intToBytes(int(vsize1)))
        vlen := t[i].Get("len").(int64)
        fmt.Printf("%x\r\n", vlen)

        if vbin0 := t[i].Get("bin0"); vbin0 != nil {
            this.BinAddress[uint(vaddress0)] = vbin0.(string)
        }
        if vbin1 := t[i].Get("bin1"); vbin1 != nil {
            this.BinAddress[uint(vaddress1)] = vbin1.(string)
        } 

        fmt.Println()
    }

    copy(outBytes[16+len(t)*36:], intToBytes(int(crc32.ChecksumIEEE(outBytes[16:16+len(t)*36])))[0:]) 
    fmt.Println(hex.EncodeToString(outBytes))
    
    ioutil.WriteFile(this.OfName, outBytes, 0666)

    return true
}

func intToBytes(n int) []byte {
  x := int32(n)
  bytesBuffer := bytes.NewBuffer([]byte{})
  binary.Write(bytesBuffer, binary.LittleEndian, x)
  return bytesBuffer.Bytes()
}
