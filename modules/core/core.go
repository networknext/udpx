package core

import (
	"fmt"
	"net"
	"os"
	"math"
	"bytes"
	"strconv"
	"hash/fnv"
	"crypto/rand"
	"encoding/binary"
)

var debugLogs bool

func init() {
	value, ok := os.LookupEnv("NEXT_DEBUG_LOGS")
	if ok && value == "1" {
		debugLogs = true
	}
}

func Error(s string, params ...interface{}) {
	fmt.Printf("error: "+s+"\n", params...)
}

func Debug(s string, params ...interface{}) {
	if debugLogs {
		fmt.Printf(s+"\n", params...)
	}
}

const (
	IPAddressNone = 0
	IPAddressIPv4 = 1
	IPAddressIPv6 = 2
	AddressSize   = 19
)

func ParseAddress(input string) *net.UDPAddr {
	address := &net.UDPAddr{}
	ip_string, port_string, err := net.SplitHostPort(input)
	if err != nil {
		address.IP = net.ParseIP(input)
		address.Port = 0
		return address
	}
	address.IP = net.ParseIP(ip_string)
	address.Port, _ = strconv.Atoi(port_string)
	return address
}

func WriteBool(data []byte, index *int, value bool) {
	if value {
		data[*index] = byte(1)
	} else {
		data[*index] = byte(0)
	}

	*index += 1
}

func WriteUint8(data []byte, index *int, value uint8) {
	data[*index] = byte(value)
	*index += 1
}

func WriteUint16(data []byte, index *int, value uint16) {
	binary.LittleEndian.PutUint16(data[*index:], value)
	*index += 2
}

func WriteUint32(data []byte, index *int, value uint32) {
	binary.LittleEndian.PutUint32(data[*index:], value)
	*index += 4
}

func WriteUint64(data []byte, index *int, value uint64) {
	binary.LittleEndian.PutUint64(data[*index:], value)
	*index += 8
}

func WriteFloat32(data []byte, index *int, value float32) {
	uintValue := math.Float32bits(value)
	WriteUint32(data, index, uintValue)
}

func WriteFloat64(data []byte, index *int, value float64) {
	uintValue := math.Float64bits(value)
	WriteUint64(data, index, uintValue)
}

func WriteString(data []byte, index *int, value string, maxStringLength uint32) {
	stringLength := uint32(len(value))
	if stringLength > maxStringLength {
		panic("string is too long!\n")
	}
	binary.LittleEndian.PutUint32(data[*index:], stringLength)
	*index += 4
	for i := 0; i < int(stringLength); i++ {
		data[*index] = value[i]
		*index++
	}
}

func WriteBytes(data []byte, index *int, value []byte, numBytes int) {
	for i := 0; i < numBytes; i++ {
		data[*index] = value[i]
		*index++
	}
}

func WriteAddress(buffer []byte, address *net.UDPAddr) {
	if address == nil {
		buffer[0] = IPAddressNone
		return
	}
	ipv4 := address.IP.To4()
	port := address.Port
	if ipv4 != nil {
		buffer[0] = IPAddressIPv4
		buffer[1] = ipv4[0]
		buffer[2] = ipv4[1]
		buffer[3] = ipv4[2]
		buffer[4] = ipv4[3]
		buffer[5] = (byte)(port & 0xFF)
		buffer[6] = (byte)(port >> 8)
	} else {
		buffer[0] = IPAddressIPv6
		copy(buffer[1:], address.IP)
		buffer[17] = (byte)(port & 0xFF)
		buffer[18] = (byte)(port >> 8)
	}
}

func ReadBool(data []byte, index *int, value *bool) bool {
	if *index+1 > len(data) {
		return false
	}

	if data[*index] > 0 {
		*value = true
	} else {
		*value = false
	}

	*index += 1
	return true
}

func ReadUint8(data []byte, index *int, value *uint8) bool {
	if *index+1 > len(data) {
		return false
	}
	*value = data[*index]
	*index += 1
	return true
}

func ReadUint16(data []byte, index *int, value *uint16) bool {
	if *index+2 > len(data) {
		return false
	}
	*value = binary.LittleEndian.Uint16(data[*index:])
	*index += 2
	return true
}

func ReadUint32(data []byte, index *int, value *uint32) bool {
	if *index+4 > len(data) {
		return false
	}
	*value = binary.LittleEndian.Uint32(data[*index:])
	*index += 4
	return true
}

func ReadUint64(data []byte, index *int, value *uint64) bool {
	if *index+8 > len(data) {
		return false
	}
	*value = binary.LittleEndian.Uint64(data[*index:])
	*index += 8
	return true
}

func ReadFloat32(data []byte, index *int, value *float32) bool {
	var intValue uint32
	if !ReadUint32(data, index, &intValue) {
		return false
	}
	*value = math.Float32frombits(intValue)
	return true
}

func ReadFloat64(data []byte, index *int, value *float64) bool {
	var uintValue uint64
	if !ReadUint64(data, index, &uintValue) {
		return false
	}
	*value = math.Float64frombits(uintValue)
	return true
}

func ReadString(data []byte, index *int, value *string, maxStringLength uint32) bool {
	var stringLength uint32
	if !ReadUint32(data, index, &stringLength) {
		return false
	}
	if stringLength > maxStringLength {
		return false
	}
	if *index+int(stringLength) > len(data) {
		return false
	}
	stringData := make([]byte, stringLength)
	for i := uint32(0); i < stringLength; i++ {
		stringData[i] = data[*index]
		*index++
	}
	*value = string(stringData)
	return true
}

func ReadBytes(data []byte, index *int, value *[]byte, bytes uint32) bool {
	if *index+int(bytes) > len(data) {
		return false
	}
	*value = make([]byte, bytes)
	for i := uint32(0); i < bytes; i++ {
		(*value)[i] = data[*index]
		*index++
	}
	return true
}

func ReadAddress(buffer []byte) *net.UDPAddr {
	addressType := buffer[0]
	switch addressType {
	case IPAddressIPv4:
		return &net.UDPAddr{IP: net.IPv4(buffer[1], buffer[2], buffer[3], buffer[4]), Port: ((int)(binary.LittleEndian.Uint16(buffer[5:])))}
	case IPAddressIPv6:
		return &net.UDPAddr{IP: buffer[1:], Port: ((int)(binary.LittleEndian.Uint16(buffer[17:])))}
	}
	return nil
}

func RandomBytes(bytes int) []byte {
	buffer := make([]byte, bytes)
	_, _ = rand.Read(buffer)
	return buffer
}

func GeneratePittle(output []byte, fromAddress []byte, fromPort uint16, toAddress []byte, toPort uint16, packetLength int) {

	var fromPortData [2]byte
	binary.LittleEndian.PutUint16(fromPortData[:], fromPort)

	var toPortData [2]byte
	binary.LittleEndian.PutUint16(toPortData[:], toPort)

	var packetLengthData [4]byte
	binary.LittleEndian.PutUint32(packetLengthData[:], uint32(packetLength))

	sum := uint16(0)

    for i := 0; i < len(fromAddress); i++ {
    	sum += uint16(fromAddress[i])
    }

    sum += uint16(fromPortData[0])
    sum += uint16(fromPortData[1])

    for i := 0; i < len(toAddress); i++ {
    	sum += uint16(toAddress[i])
    }

    sum += uint16(toPortData[0])
    sum += uint16(toPortData[1])

    sum += uint16(packetLengthData[0])
    sum += uint16(packetLengthData[1])
    sum += uint16(packetLengthData[2])
    sum += uint16(packetLengthData[3])

	var sumData [2]byte
	binary.LittleEndian.PutUint16(sumData[:], sum)

    output[0] = 1 | ( sumData[0] ^ sumData[1] ^ 193 );
    output[1] = 1 | ( ( 255 - output[0] ) ^ 113 );
}

func GenerateChonkle(output []byte, magic []byte, fromAddressData []byte, fromPort uint16, toAddressData []byte, toPort uint16, packetLength int) {

	var fromPortData [2]byte
	binary.LittleEndian.PutUint16(fromPortData[:], fromPort)

	var toPortData [2]byte
	binary.LittleEndian.PutUint16(toPortData[:], toPort)

	var packetLengthData [4]byte
	binary.LittleEndian.PutUint32(packetLengthData[:], uint32(packetLength))

	hash := fnv.New64a()
	hash.Write(magic)
	hash.Write(fromAddressData)
	hash.Write(fromPortData[:])
	hash.Write(toAddressData)
	hash.Write(toPortData[:])
	hash.Write(packetLengthData[:])
	hashValue := hash.Sum64()

	var data [8]byte
	binary.LittleEndian.PutUint64(data[:], uint64(hashValue))

    output[0] = ( ( data[6] & 0xC0 ) >> 6 ) + 42
    output[1] = ( data[3] & 0x1F ) + 200
    output[2] = ( ( data[2] & 0xFC ) >> 2 ) + 5
    output[3] = data[0]
    output[4] = ( data[2] & 0x03 ) + 78
    output[5] = ( data[4] & 0x7F ) + 96
    output[6] = ( ( data[1] & 0xFC ) >> 2 ) + 100
    if ( data[7] & 1 ) == 0 { 
    	output[7] = 79
    } else { 
    	output[7] = 7 
    }
    if ( data[4] & 0x80 ) == 0 {
    	output[8] = 37
    } else { 
    	output[8] = 83
    }
    output[9] = ( data[5] & 0x07 ) + 124
    output[10] = ( ( data[1] & 0xE0 ) >> 5 ) + 175
    output[11] = ( data[6] & 0x3F ) + 33
    value := ( data[1] & 0x03 ); 
    if value == 0 { 
    	output[12] = 97
    } else if value == 1 { 
    	output[12] = 5
    } else if value == 2 { 
    	output[12] = 43
    } else { 
    	output[12] = 13
    }
    output[13] = ( ( data[5] & 0xF8 ) >> 3 ) + 210
    output[14] = ( ( data[7] & 0xFE ) >> 1 ) + 17
}

func BasicPacketFilter(data []byte, packetLength int) bool {

    if packetLength < 18 {
        return false
    }

    if data[0] < 0x01 || data[0] > 0x63 {
        return false
    }

    if data[1] < 0x2A || data[1] > 0x2D {
        return false
    }

    if data[2] < 0xC8 || data[2] > 0xE7 {
        return false
    }

    if data[3] < 0x05 || data[3] > 0x44 {
        return false
    }

    if data[5] < 0x4E || data[5] > 0x51 {
        return false
    }

    if data[6] < 0x60 || data[6] > 0xDF {
        return false
    }

    if data[7] < 0x64 || data[7] > 0xE3 {
        return false
    }

    if data[8] != 0x07 && data[8] != 0x4F {
        return false
    }

    if data[9] != 0x25 && data[9] != 0x53 {
        return false
    }
    
    if data[10] < 0x7C || data[10] > 0x83 {
        return false
    }

    if data[11] < 0xAF || data[11] > 0xB6 {
        return false
    }

    if data[12] < 0x21 || data[12] > 0x60 {
        return false
    }

    if data[13] != 0x61 && data[13] != 0x05 && data[13] != 0x2B && data[13] != 0x0D {
        return false
    }

    if data[14] < 0xD2 || data[14] > 0xF1 {
        return false
    }

    if data[15] < 0x11 || data[15] > 0x90 {
        return false
    }

    return true
}

func AdvancedPacketFilter(data []byte, magic []byte, fromAddress []byte, fromPort uint16, toAddress []byte, toPort uint16, packetLength int) bool {
    if packetLength < 18 {
        return false;
    }
    var a [15]byte
    var b [2]byte
    GenerateChonkle(a[:], magic, fromAddress, fromPort, toAddress, toPort, packetLength)
    GeneratePittle(b[:], fromAddress, fromPort, toAddress, toPort, packetLength)
    if bytes.Compare(a[0:15], data[1:16]) != 0 {
        return false
    }
    if bytes.Compare(b[0:2], data[packetLength-2:packetLength]) != 0 {
        return false
    }
    return true;
}

func GetAddressData(address *net.UDPAddr, addressData []byte, addressPort *uint16, addressBytes *int) {

	// todo
	*addressPort = 0
	*addressBytes = 0

	/*
    next_assert( address );
    if ( address->type == NEXT_ADDRESS_IPV4 )
    {
        address_data[0] = address->data.ipv4[0];
        address_data[1] = address->data.ipv4[1];
        address_data[2] = address->data.ipv4[2];
        address_data[3] = address->data.ipv4[3];
        *address_bytes = 4;
    }
    else if ( address->type == NEXT_ADDRESS_IPV6 )
    {
        for ( int i = 0; i < 8; ++i )
        {
            address_data[i*2]   = address->data.ipv6[i] >> 8;
            address_data[i*2+1] = address->data.ipv6[i] & 0xFF;
        }
        *address_bytes = 16;
    }
    else
    {
        *address_bytes = 0;
    }
    *address_port = address->port;
    */
}
