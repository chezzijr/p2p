package connection

type BitField []byte

func (bf BitField) HasPiece(index int) bool {
    byteIndex := index / 8
    offset := index % 8
    if byteIndex < 0 || byteIndex >= len(bf) {
        return false
    }
    return bf[byteIndex] >> uint(7 - offset) & 1 != 0
}

func (bf BitField) SetPiece(index int) {
    byteIndex := index / 8
    offset := index % 8
    if byteIndex < 0 || byteIndex >= len(bf) {
        return
    }
    bf[byteIndex] |= 1 << uint(7 - offset)
}

func NewBitField(size int) BitField {
    return make(BitField, (size+7)/8)
}
