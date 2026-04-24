package bzip2

// bzip2 uses CRC-32 with polynomial 0x04C11DB7, processed MSB-first with
// no input/output reflection. The stored block CRC is the bitwise
// complement of the final accumulator.

var crcTable [256]uint32

func init() {
	for i := range crcTable {
		r := uint32(i) << 24
		for range 8 {
			if r&0x80000000 != 0 {
				r = (r << 1) ^ 0x04c11db7
			} else {
				r <<= 1
			}
		}
		crcTable[i] = r
	}
}

func blockCRC(data []byte) uint32 {
	crc := uint32(0xFFFFFFFF)
	for _, b := range data {
		crc = (crc << 8) ^ crcTable[((crc>>24)^uint32(b))&0xFF]
	}
	return ^crc
}
