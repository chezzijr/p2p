package connection

type BitField []byte

func (bf BitField) HasPiece(index int) bool {
	byteIndex := index / 8
	offset := index % 8
	if byteIndex < 0 || byteIndex >= len(bf) {
		return false
	}
	return bf[byteIndex]>>uint(7-offset)&1 != 0
}

func (bf BitField) SetPiece(index int) {
	byteIndex := index / 8
	offset := index % 8
	if byteIndex < 0 || byteIndex >= len(bf) {
		return
	}
	bf[byteIndex] |= 1 << uint(7-offset)
}

func (bf BitField) NumPieces() int {
	// count number of 1 bits
	count := 0
	// https://stackoverflow.com/questions/45520191/how-count-how-many-one-bit-have-in-byte-in-golang
	for _, v := range bf {
		v = (v & 0x55) + ((v >> 1) & 0x55)
		v = (v & 0x33) + ((v >> 2) & 0x33)
		count += int((v + (v >> 4)) & 0xF)
	}
	return count
}

func NewBitField(size int) BitField {
	return make(BitField, (size+7)/8)
}
