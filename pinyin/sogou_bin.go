package pinyin

import (
	"bytes"
	"io/ioutil"

	encoder "github.com/cxcn/dtool/encoders"
	. "github.com/cxcn/dtool/utils"
)

type key struct {
	dictType    int // 2 字节
	dataTypeLen int // 2 字节
	dataType    []int
	attrIdx     int // 4 字节
	keyDataIdx  int // 4 字节
	dataIdx     int // 4 字节
	v6          int // 4 字节
}

// 全是 4 字节
type attr struct {
	count  int
	a2     int
	dataId int
	b2     int
}

// 全是 4 字节
type header struct {
	offset       int // 偏移量
	dataSize     int
	usedDataSize int
}

func (h *header) parse(r *bytes.Reader) {
	h.offset = ReadUint32(r)
	h.dataSize = ReadUint32(r)
	h.usedDataSize = ReadUint32(r)
}

func ParseSogouBin(filename string) WpfDict {
	data, _ := ioutil.ReadFile(filename)
	r := bytes.NewReader(data)
	ret := make(WpfDict, 0, r.Len()>>8)
	// var tmp []byte

	// fileChksum := ReadUint32(r)
	// size1 := ReadUint32(r)
	r.Seek(8, 1)
	keyLen := ReadUint32(r)
	attrLen := ReadUint32(r)
	aintLen := ReadUint32(r)
	// fmt.Println(fileChksum, size1, keyLen, attrLen, aintLen)

	keys := make([]key, 0, 1)
	for i := 0; i < keyLen; i++ {
		var k key
		k.dictType = ReadUint16(r)
		k.dataTypeLen = ReadUint16(r)
		k.dataType = make([]int, 0, 1)
		for j := 0; j < k.dataTypeLen; j++ {
			dataType := ReadUint16(r)
			k.dataType = append(k.dataType, dataType)
		}

		k.attrIdx = ReadUint32(r)
		k.keyDataIdx = ReadUint32(r)
		k.dataIdx = ReadUint32(r)
		k.v6 = ReadUint32(r)

		keys = append(keys, k)
	}

	attrs := make([]attr, 0, 1)
	for i := 0; i < attrLen; i++ {
		var a attr
		a.count = ReadUint32(r)
		a.a2 = ReadUint32(r)
		a.dataId = ReadUint32(r)
		a.b2 = ReadUint32(r)
		attrs = append(attrs, a)
	}

	aints := make([]int, 0, 1)
	for i := 0; i < aintLen; i++ {
		aint := ReadUint32(r)
		aints = append(aints, aint)
	}

	// fmt.Printf("keys %+v\n", keys)
	// fmt.Printf("attrs %+v\n", attrs)
	// fmt.Printf("aints %+v\n", aints)

	ud := newUsrDict()
	ud.keys = keys
	ud.attrs = attrs
	ud.aints = aints

	// b2Ver := ReadUint32(r)
	// b2Format := ReadUint32(r)
	// size2 := ReadUint32(r)
	r.Seek(12, 1)
	// fmt.Println(b2Ver, b2Format, size2)

	hiLen := ReadUint32(r)
	haLen := ReadUint32(r)
	hsLen := ReadUint32(r)
	// fmt.Println(hiLen, haLen, hsLen)

	for i := 0; i < hiLen; i++ {
		var h header
		h.parse(r)
		ud.headerIdxs = append(ud.headerIdxs, h)
	}
	for i := 0; i < haLen; i++ {
		var h header
		h.parse(r)
		ud.headerAttrs = append(ud.headerAttrs, h)
	}
	for i := 0; i < hsLen; i++ {
		var h header
		h.parse(r)
		ud.dataStore = append(ud.dataStore, h)
	}
	// fmt.Printf("headerIdxs %+v\n", ud.headerIdxs)
	// fmt.Printf("headerAttrs %+v\n", ud.headerAttrs)
	// fmt.Printf("dataStore %+v\n", ud.dataStore)

	p2Idx := len(data) - 4*5
	p3Idx := len(data) - 4*4
	p2 := BytesToInt(data[p2Idx : p2Idx+4])
	p3 := BytesToInt(data[p3Idx : p3Idx+4])
	// fmt.Println(p2, p3)

	ud.init()
	preOffset := r.Size() - int64(r.Len())
	d := ud.getData(r)
	for i := 0; i < len(d)/2; i++ {

		a, b := d[2*i], d[2*i+1]
		offset := ud.dataStore[ud.keys[0].keyDataIdx].offset + a
		// fmt.Printf("a: %v, b: %v, offset: %v\t", a, b, offset)

		var wordInfo attrWordData
		r.Seek(int64(b), 0)
		wordInfo.parse(r)
		// fmt.Printf("wordInfo: %v", wordInfo)
		// GetWordData
		attrId := ud.keys[0].attrIdx
		dataId := ud.attrs[attrId].dataId
		offset = int(preOffset) + ud.dataStore[dataId].offset + wordInfo.offset
		// fmt.Printf("offset: %v\n", offset)
		// DecryptWordsEx
		word := decryptWordsEx(r, offset, wordInfo.p1, p2, p3)
		pinyin := encoder.GetPinyin(word)
		ret = append(ret, WordPyFreq{word, pinyin, wordInfo.freq})
		// fmt.Printf("word: %v\tcode: %v\tfreq: %v\n", word, codes, wordInfo.freq)
	}
	return ret
}

func decryptWordsEx(r *bytes.Reader, offset, p1, p2, p3 int) string {
	k1 := (p1 + p2) << 2
	k2 := (p1 + p3) << 2
	xk := (k1 + k2) & 0xFFFF
	r.Seek(int64(offset), 0)
	n := ReadUint16(r) / 2
	decWords := make([]byte, 0, 1)
	for i := 0; i < n; i++ {
		shift := p2 % 8
		ch := ReadUint16(r)
		dch := (ch<<(16-(shift%8)) | (ch >> shift)) & 0xFFFF
		dch ^= xk
		if dch > 0x10000 {
			print(dch)
		}
		decWords = append(decWords, byte(dch%0x100), byte(dch>>8))
	}
	ret, _ := Decode(decWords, "utf16")
	return ret
}

type attrWordData struct {
	offset int
	freq   int
	aflag  int
	i8     int
	p1     int
	iE     int
}

func (a *attrWordData) parse(r *bytes.Reader) {
	a.offset = ReadUint32(r)
	a.freq = ReadUint16(r)
	a.aflag = ReadUint16(r)
	a.i8 = ReadUint32(r)
	a.p1 = ReadUint16(r)
	a.iE = ReadUint32(r) // always zero
	_ = ReadUint32(r)    // next offset
}

type usrDict struct {
	keys        []key
	attrs       []attr
	aints       []int
	headerIdxs  []header
	headerAttrs []header
	dataStore   []header

	dataTypeSize []int
	attrSize     []int
	baseHashSize []int
	keyHashSize  []int
	aflag        bool
}

var keyDataTypeSize = []int{4, 1, 1, 2, 1, 2, 2, 4, 4, 8, 4, 4, 4, 0, 0, 0}
var dataTypeHashSize = []int{0, 27, 414, 512, -1, -1, 512, 0}

func newUsrDict() *usrDict {
	ud := new(usrDict)
	ud.headerIdxs = make([]header, 0, 1)
	ud.headerAttrs = make([]header, 0, 1)
	ud.dataStore = make([]header, 0, 1)

	ud.dataTypeSize = make([]int, 0, 1)
	// ud.attrSize = make([]int, 0, 1)
	ud.baseHashSize = make([]int, 0, 1)
	ud.keyHashSize = make([]int, 10)
	ud.keyHashSize[0] = 500
	// fmt.Printf("newUsrDict%+v\n", ud)
	return ud
}

func (ud *usrDict) init() {
	ud.attrSize = make([]int, len(ud.attrs))

	for i, k := range ud.keys {
		size := (k.dictType >> 2) & 4
		maskedType := k.dictType & 0xFFFFFF8F
		// hash item
		if ud.keyHashSize[i] > 0 {
			ud.baseHashSize = append(ud.baseHashSize, ud.keyHashSize[i])
		} else {
			ud.baseHashSize = append(ud.baseHashSize, dataTypeHashSize[maskedType])
		}
		// dataType size
		attrCount := ud.attrs[k.attrIdx].count
		// non-attr data size
		nonAttrCount := len(k.dataType) - attrCount
		for j := 0; j < nonAttrCount; j++ {
			if j > 0 || maskedType != 4 {
				size += keyDataTypeSize[k.dataType[i]]
			}
		}
		if k.dictType&0x60 > 0 {
			size += 4
		}
		size += 4
		ud.dataTypeSize = append(ud.dataTypeSize, size)
		// attr data size
		attrSize := 0
		for j := nonAttrCount; j < len(k.dataType); j++ {
			attrSize += keyDataTypeSize[k.dataType[j]]
		}
		if (k.dictType & 0x40) == 0 {
			attrSize += 4
		}
		ud.attrSize[k.attrIdx] = attrSize
		// ???
		if ud.attrs[k.attrIdx].b2 == 0 {
			ud.aflag = true
		}
	}
	// fmt.Printf("init UsrDict%+v\n", ud)
}

func (ud *usrDict) getData(r *bytes.Reader) []int {

	ret := make([]int, 0, 0xff)
	keyId := 0
	theKey := ud.keys[keyId]

	// hashStoreBase := ud.getHashStore(keyId, theKey.dictType&0xFFFFFF8F)
	headerAttr := ud.headerAttrs[theKey.attrIdx]
	var attrCount int

	if headerAttr.usedDataSize == 0 {
		attrCount = headerAttr.dataSize
	} else {
		attrCount = headerAttr.usedDataSize
	}
	hashStoreCount := ud.baseHashSize[keyId]
	// fmt.Println("getData", hashStoreBase, hashStoreCount)

	preOffset := r.Size() - int64(r.Len())
	for i := 0; i < hashStoreCount; i++ {
		r.Seek(preOffset+int64(8*i), 0)
		hashStoreOffset := ReadUint32(r)
		hashStoreCount := ReadUint32(r)

		// fmt.Printf("hashstore [ offset: {%v}, count: {%v} ]\n", hashStoreOffset, hashStoreCount)
		for j := 0; j < hashStoreCount; j++ {

			attrOffset := int(preOffset) + ud.headerIdxs[keyId].offset + hashStoreOffset + ud.dataTypeSize[keyId]*j
			offset := attrOffset + ud.dataTypeSize[keyId] - 4
			r.Seek(int64(offset), 0)
			offset = ReadUint32(r)
			// fmt.Printf("\tattrOffset, %d %d\n", attrOffset, offset)
			for k := 0; k < attrCount; k++ {

				attr2Offset := int(preOffset) + ud.headerAttrs[ud.keys[keyId].attrIdx].offset + offset
				ret = append(ret, attrOffset, attr2Offset)

				offset = attr2Offset + ud.attrSize[theKey.attrIdx] - 4
				r.Seek(int64(offset), 0)
				offset = ReadUint32(r)
				// fmt.Printf("\tattr2Offset, %d ,newOffset, %d \n", attr2Offset, offset)
				if offset == 0xFFFFFFFF {
					break
				}
			}
		}
	}
	// fmt.Println(ret)
	return ret
}

func (ud *usrDict) getHashStore(idx, dataType int) int {
	if idx < 0 || dataType > 6 || idx > len(ud.headerIdxs) {
		panic("getHashStore error")
	}
	offset := ud.headerIdxs[idx].offset
	// assert index_offset >= 0
	size := ud.baseHashSize[idx]
	offset = offset - 8*size
	// assert offset >= 0
	return offset
}
