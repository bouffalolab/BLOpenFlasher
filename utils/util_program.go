package utils

import (
    "fmt"
    "os"
    "io"
    "reflect"
    "errors"
    "time"
    "strings"
//  "sync"
    "strconv"
    "bytes"
    "crypto/sha256"
    "encoding/hex"
//  "github.com/tarm/serial"
    "github.com/albenik/go-serial"
//  "github.com/mikepb/go-serial"
)

func sendCmd(name string, s *serial.Port, p []byte, cmd string) bool {
    nw, err:= s.Write(p)
    if err != nil {
        fmt.Println(name+" Failed to write ",cmd, ", length  ", nw)
        return false
    }
    //fmt.Println(time.Now())
    //fmt.Println(name + " -> "+hex.EncodeToString(p));
    //fmt.Printf("%v %v==%v\r\n", name, nw, len(p))

    return true
}

func recvRes(name string, s *serial.Port, l int, try int) []byte{
    max := try
    offset := 0
    rbuf := make([]byte, 256)
    for {
        nr, err := s.Read(rbuf[offset:])
        if err != nil {
            fmt.Println(name+" Failed to read [Res]")
            break
        } else {
            try--
            if try == 0 {
                break;
            }
            //fmt.Print(nr)
            //fmt.Print(rbuf[:offset])
            offset += nr
            if nr > 0 {
                //fmt.Println(rbuf)
            }
            if offset >= l {
                break
            }
        }
    }
    //fmt.Println(time.Now())
    fmt.Printf(name+" %v:%v <-", max, try)
    //fmt.Println(rbuf[:offset])
    fmt.Println(hex.EncodeToString(rbuf[:offset]))
    return rbuf[:offset]
}


type config struct{
    name string
    uart *serial.Port
    readTimeout int
    romBaud int
    loaderBaud int
    resetCount int

    f *os.File
    loaderBin string
    bins []string
    binIndex int
    startAddr int
    curAddr int
    fileSize int
    sha256 []byte
    eraseTimeout int

    resData []byte
}


func DynamicMethod(object interface{}, methodName string, args ...interface{}) ([]reflect.Value, error){
    inputs := make([]reflect.Value, len(args))
    for i, _ := range args {
        inputs[i] = reflect.ValueOf(args[i])
    }

    method := reflect.ValueOf(object).MethodByName(methodName)
    if method.IsValid(){
        return method.Call(inputs), nil
    } else {
        return nil, errors.New(methodName+" can not be located!")
    }
}

func (this *config)MulTryCom(cmdName string, send []byte, retry int, recv int, timeout int) bool{
    for ; retry > 0 ; retry-- {
        if sendCmd(this.name, this.uart, send, cmdName){
            resData := recvRes(this.name, this.uart, recv, timeout/this.readTimeout+1)
            if len(resData) == recv && string(resData[:2]) == "OK" {
                this.resData = resData
                return true
            }
        }
    }
    return false
}

var resetCounter int

func (this *config)ConfigReset() string{
    var err error 
    this.f.Close()
    this.f, err = os.Open(this.loaderBin)
    this.binIndex = 0
    this.resetCount = 0

    if err != nil {
        return "ErrorLoaderBin"
    } 

    this.uart.ResetInputBuffer()
    this.uart.ResetOutputBuffer()

    return "CmdReset"
}

func (this *config)CmdReset() string{
    this.resetCount++
    if this.resetCount > 3 {
        return "ErrorShakeHand"
    }
//Module
    fmt.Println(this.name+"---RESET Module---")
    this.uart.SetRTS(true)
    //fmt.Println(this.name+" SetRTS true")
    time.Sleep(time.Millisecond * 50)

    err := this.uart.SetDTR(true)
    //fmt.Println(this.name+" SetDTR true")
    if err != nil {
        fmt.Println(this.name+" Failed to SetDTR")
    }
    time.Sleep(time.Millisecond * 50)
    err = this.uart.SetDTR(false)
    //fmt.Println(this.name+" SetDTR false")
    if err != nil {
        fmt.Println(this.name+" Failed to SetDTR")
    }
    time.Sleep(time.Millisecond * 50)
    this.uart.SetRTS(false)
    //fmt.Println(this.name+" SetRTS false")


    time.Sleep(time.Millisecond * 50)

/*
    fmt.Println("---RESET SocketBoard---")
    this.uart.SetDTR(true)
    this.uart.SetRTS(true)
    time.Sleep(time.Millisecond * 200)
    this.uart.SetRTS(false)
    time.Sleep(time.Millisecond * 5)
    this.uart.SetRTS(true)
    time.Sleep(time.Millisecond * 100)

    this.uart.SetDTR(true)
    time.Sleep(time.Millisecond * 10)
    this.uart.SetDTR(false)
    this.uart.SetRTS(false)
    time.Sleep(time.Millisecond * 200)
    this.uart.SetRTS(true)
    time.Sleep(time.Millisecond * 5)
    this.uart.SetRTS(false)
    time.Sleep(time.Millisecond * 100)
    this.uart.SetRTS(true)
    time.Sleep(time.Millisecond * 5)
    this.uart.SetRTS(false)
*/

    this.uart.Reconfigure(serial.WithBaudrate(this.romBaud))
    this.uart.ResetInputBuffer()
    this.uart.ResetOutputBuffer()

    return "CmdShakeHand"
}

func (this *config)CmdShakeHand() string{
    cmdLength := 7 * this.romBaud / 10000
    cmdPacket := make([]byte, cmdLength)
    for i:=0; i<cmdLength; i++ {
        cmdPacket[i] = 0x55
    }
    if this.MulTryCom("ShakeHand", cmdPacket, 3 , 2, 200) {
        return "CmdBootInfo"
    } else {
        return "CmdReset"
    }
}

func (this *config)CmdBootInfo() string{
    cmdPacket := []byte{0x10, 0x00, 0x00, 0x00}
    if this.MulTryCom("BootInfo", cmdPacket, 3 , 24, 200) {
        return "CmdBootHeader"
    } else {
        return "CmdReset"
    }
}

func (this *config)CmdBootHeader() string{
    cmdPacket := make([]byte, 180)
    cmdPacket[0] = 0x11
    cmdPacket[1] = 0x00
    cmdPacket[2] = 0xb0
    cmdPacket[3] = 0x00
    this.f.Read(cmdPacket[4:])

    if this.MulTryCom("BootHeader", cmdPacket, 3 , 2, 200) {
        return "CmdSegHeader"
    } else {
        return "ConfigReset"
    }
}

func (this *config)CmdSegHeader() string{
    cmdPacket := make([]byte, 20)
    cmdPacket[0] = 0x17
    cmdPacket[1] = 0x00
    cmdPacket[2] = 0x10
    cmdPacket[3] = 0x00
    this.f.Read(cmdPacket[4:])

    if this.MulTryCom("SegHeader", cmdPacket, 3 , 20, 200) {
        return "CmdSegData"
    } else {
        return "ConfigReset"
    }
}

func (this *config)CmdSegData() string{
    cmdPacket := make([]byte, 4+2048)
    n, _ :=this.f.Read(cmdPacket[4:])
    if n == 0 {
        return "CmdCheckImage"
    }
    //fmt.Printf("send %d\r\n", n)
    cmdPacket[0] = 0x18
    cmdPacket[1] = 0x00
    cmdPacket[2] = byte(n & 0xff)
    cmdPacket[3] = byte((n & 0xff00) >> 8)

    if this.MulTryCom("SegData", cmdPacket[:n+4], 3 , 2, 200) {
        return "CmdSegData"
    } else {
        return "ConfigReset"
    }
}

func (this *config)CmdCheckImage() string{
    cmdPacket := []byte{0x19, 0x00, 0x00, 0x00}

    if this.MulTryCom("CheckImage", cmdPacket, 3 , 2, 200) {
        return "CmdRunImage"
    } else {
        return "ConfigReset"
    }
}

func (this *config)CmdRunImage() string{
    cmdPacket := []byte{0x1a, 0x00, 0x00, 0x00}

    if this.MulTryCom("RunImage", cmdPacket, 3 , 2, 200) {
        return "CmdReshake"
    } else {
        return "ConfigReset"
    }
}

func (this *config)CmdReshake() string{
    this.uart.Reconfigure(serial.WithBaudrate(this.loaderBaud))
    time.Sleep(time.Millisecond * 200)
    cmdLength := 7 * this.loaderBaud / 10000
    cmdPacket := make([]byte, cmdLength)
    for i:=0; i<cmdLength; i++ {
        cmdPacket[i] = 0x55
    }

    if this.MulTryCom("Reshake", cmdPacket, 3 , 2, 200) {
        return "CmdLoadFile"
    } else {
        return "ConfigReset"
    }
}

func (this *config)CmdLoadFile() string{
    var err error
    var addr64 int64
    this.f.Close()

    if this.binIndex >= len(this.bins) {
        return "CmdProgramFinish"
    }
    
    fwaddr := strings.Split(this.bins[this.binIndex], "@")

    fmt.Printf("%v %v\r\n", this.name, fwaddr)
    this.f, err = os.Open(fwaddr[0])
    if err != nil {
        fmt.Println(err)
        return ("ErrorOpenFile"+fwaddr[0])
    }

    addr64, err = strconv.ParseInt(fwaddr[1], 0, 64)
    if err != nil {
        fmt.Println(err)
        return ("ErrorConverStrToInt "+fwaddr[1])
    }
    this.startAddr = int(addr64)
    this.curAddr = this.startAddr

    hash := sha256.New()
    flength, err := io.Copy(hash, this.f)
    if err != nil {
        fmt.Println(this.name+" Failed to sh256 "+fwaddr[0])
        return ("ErrorCalculateSha256")
    }
    this.sha256 = hash.Sum(nil)
    this.fileSize = int(flength)

    this.f.Seek(0, os.SEEK_SET)
    if this.fileSize == 0 {
        this.binIndex++
        return "CmdLoadFile"
    } else {
        return "CmdEraseFlash"
    }
}

/*
func (this *config)CmdEraseChip() string{
    cmdPacket := []byte{0x3c, 0x00, 0x00, 0x00}

    return "ConfigReset"
}
*/
func (this *config)CmdEraseFlash() string{
    cmdErase := make([]byte, 12)
    cmdErase[0] = 0x30
    cmdErase[2] = 0x08
    cmdErase[3] = 0x00
    cmdErase[4] = byte(this.startAddr & 0xff)
    cmdErase[5] = byte((this.startAddr>>8) & 0xff)
    cmdErase[6] = byte((this.startAddr>>16) & 0xff)
    cmdErase[7] = byte((this.startAddr>>24) & 0xff)
    cmdErase[8] = byte((this.startAddr+this.fileSize) & 0xff)
    cmdErase[9] = byte(((this.startAddr+this.fileSize)>>8) & 0xff)
    cmdErase[10] = byte(((this.startAddr+this.fileSize)>>16) & 0xff)
    cmdErase[11] = byte(((this.startAddr+this.fileSize)>>24) & 0xff)

    crc := 0
    for i:=2; i<12; i++ {
        crc += int(cmdErase[i])
    }
    cmdErase[1] = byte(crc)

    if this.MulTryCom("EraseFlash", cmdErase, 2 , 2, this.eraseTimeout) {
        return "CmdProgramFlash"
    } else {
        return "ErrorEraseFlash"
    }
}

func (this *config)CmdProgramFlash() string{
    cmdPacket := make([]byte, 4+4+8192)
    l, _ := this.f.Read(cmdPacket[8:])

    //fmt.Println(l)
    cmdPacket[0] = 0x31
    cmdPacket[2] = byte((l+4) & 0xff)
    cmdPacket[3] = byte(((l+4)>>8) & 0xff)
    cmdPacket[4] = byte(this.curAddr & 0xff)
    cmdPacket[5] = byte((this.curAddr>>8) & 0xff)
    cmdPacket[6] = byte((this.curAddr>>16) & 0xff)
    cmdPacket[7] = byte((this.curAddr>>24) & 0xff)
    crc := 0
    for i:=2; i<l+8 ;i++ {
        crc += int(cmdPacket[i])
    }
    cmdPacket[1] = byte(crc)

    if this.MulTryCom("ProgramFlash", cmdPacket[:8+l], 3 , 2, 400) {
        this.curAddr += l
        if this.curAddr < this.startAddr + this.fileSize {
            return "CmdProgramFlash"
        } else {
            return "CmdProgramOK"
        }
    } else {
        return "ErrorProgramFLash"
    }
}

func (this *config)CmdProgramOK() string{
    cmdCheck :=  []byte{0x3A, 0x00, 0x00, 0x00}

    if this.MulTryCom("EraseFlash", cmdCheck, 3, 2, 400) {
        return "CmdSha256"
    } else {
        return "ErrorProgramOK"
    }
}

func (this *config)CmdSha256() string{
    cmdSha256 := make([]byte, 12)
    cmdSha256[0] = 0x3D
    cmdSha256[2] = 0x08
    cmdSha256[3] = 0x00
    cmdSha256[4] = byte(this.startAddr & 0xff)
    cmdSha256[5] = byte(this.startAddr>>8 & 0xff)
    cmdSha256[6] = byte(this.startAddr>>16 & 0xff)
    cmdSha256[7] = byte(this.startAddr>>24 & 0xff)
    cmdSha256[8] = byte(this.fileSize & 0xff)
    cmdSha256[9] = byte(this.fileSize>>8 & 0xff)
    cmdSha256[10] = byte(this.fileSize>>16 & 0xff)
    cmdSha256[11] = byte(this.fileSize>>24 & 0xff)
    crc := 0
    for i:=2; i<12 ;i++ {
        crc += int(cmdSha256[i])
    }
    cmdSha256[1] = byte(crc)

    if this.MulTryCom("Sha256", cmdSha256, 3 , 36, 200) {
        if bytes.Equal(this.resData[4:], this.sha256){
            this.binIndex++
            return "CmdLoadFile"
        } else {
            return "ErrorVerifySha256"
        }
    } else {
        return "ErrorSha256"
    }
}

func (this *config)CmdProgramFinish() string{
    this.uart.Reconfigure(serial.WithBaudrate(this.loaderBaud))
    time.Sleep(time.Millisecond * 200)
    cmdLength := 7 * this.loaderBaud / 10000
    cmdPacket := make([]byte, cmdLength)
    for i:=0; i<cmdLength; i++ {
        cmdPacket[i] = 0x55
    }
    fmt.Println("RESHAKE............")
    for{
        if this.MulTryCom("ProgramFinish", cmdPacket, 3 , 2, 1000) {
            return "CmdProgramFinish"
        } else {
            return "CmdProgramFinish"
        }
        time.Sleep(time.Second*1)
    }
}

func StartProgram(name string,  uart *serial.Port, baudrom int, eflashloader string, baudloader int, bins []string, erasetimeout int) bool{
    var gConfig config

    floader, err := os.Open(eflashloader)
    if err != nil {
        fmt.Println(name+":Failed to open "+eflashloader)
        return false
    }
    gConfig.name = name
    gConfig.uart = uart
    gConfig.readTimeout = 15
    gConfig.romBaud = baudrom
    gConfig.loaderBaud = baudloader
    gConfig.loaderBin = eflashloader
    gConfig.f = floader
    gConfig.bins = bins
    gConfig.startAddr = 0
    gConfig.curAddr = 0
    gConfig.fileSize = 0
    gConfig.eraseTimeout = erasetimeout

    if gConfig.uart != nil {
        gConfig.uart.Reconfigure(serial.WithReadTimeout(gConfig.readTimeout))
        gConfig.uart.Close()
    }

    gConfig.uart, err = serial.Open(name)
    if err != nil {
        fmt.Printf("%v : Open Error\r\n", name)
        fmt.Println(err)
        return false
    }
    gConfig.uart.ResetInputBuffer()

    gConfig.uart.Reconfigure(serial.WithReadTimeout(gConfig.readTimeout))

    defer gConfig.uart.Close()
    defer gConfig.f.Close()

    ret, err := DynamicMethod(&gConfig, "ConfigReset")
    if err != nil {
        fmt.Println(ret)
    } else {
        fmt.Println(ret[0])
    }

    for{

        if "CmdProgramFinish" == ret[0].String(){
            fmt.Println(gConfig.name+" -----Success-----")
            return true
        }
        
        ret, err = DynamicMethod(&gConfig, ret[0].String())
        if err == nil {
            fmt.Println(gConfig.name+" Next to "+ret[0].String())
        } else {
            fmt.Println(err)
            fmt.Println(gConfig.name+" -----Failure-----")

            return false
        }
    }
    return false
}
