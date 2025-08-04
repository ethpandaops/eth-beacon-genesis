package beaconutils

import "encoding/binary"

func UintToBytes(data any) []byte {
	var res []byte

	switch d := data.(type) {
	case uint64:
		res = make([]byte, 8)
		binary.LittleEndian.PutUint64(res, d)
	case uint32:
		res = make([]byte, 4)
		binary.LittleEndian.PutUint32(res, d)
	case uint16:
		res = make([]byte, 2)
		binary.LittleEndian.PutUint16(res, d)
	case uint8:
		res = []byte{d}
	}

	return res
}

func BytesToUint(data []byte) uint64 {
	switch len(data) {
	case 1:
		return uint64(data[0])
	case 2:
		return uint64(binary.LittleEndian.Uint16(data))
	case 4:
		return uint64(binary.LittleEndian.Uint32(data))
	case 8:
		return binary.LittleEndian.Uint64(data)
	default:
		return 0
	}
}
