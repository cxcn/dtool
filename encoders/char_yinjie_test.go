package encoder

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"testing"

	. "github.com/cxcn/dtool/utils"
)

// 处理原始单字拼音表
func TestGenCharYinjieMap(t *testing.T) {
	f, err := os.Open("own/src_char_yinjie.txt")
	if err != nil {
		log.Panic(err)
	}
	rd, _ := DecodeIO(f)

	type orderCodes struct {
		order int
		codes []string
	}
	charMap := make(map[rune]*orderCodes)
	var buf bytes.Buffer
	scan := bufio.NewScanner(rd)
	for order := 0; scan.Scan(); order++ {
		entry := strings.Split(scan.Text(), "\t")
		if len(entry) < 2 {
			continue
		}
		char := []rune(entry[0])[0]
		if _, ok := charMap[char]; !ok {
			charMap[char] = &orderCodes{order, entry[1:]}
			continue
		}
		charMap[char].codes = append(charMap[char].codes, entry[1:]...)
	}

	type ocw struct {
		order int
		codes []string
		word  rune
	}
	ocwSli := make([]ocw, 0, len(charMap))
	for k, v := range charMap {
		ocwSli = append(ocwSli, ocw{v.order, RmRepeat(v.codes), k})
	}
	sort.Slice(ocwSli, func(i, j int) bool {
		return ocwSli[i].order < ocwSli[j].order
	})
	for _, v := range ocwSli {
		buf.WriteRune(v.word)
		for _, vv := range v.codes {
			buf.WriteByte('\t')
			// buf.WriteByte('\'')
			buf.WriteString(vv)
		}
		buf.WriteString(LineBreak)
	}
	ioutil.WriteFile("assets/char_yinjie.txt", buf.Bytes(), 0777)
}
