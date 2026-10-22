// Independently verify struct-packed relocation fixtures using Go debug/elf types.
// Run from object: go run elf/tests/data/check_relocations.go
package main

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"os"
)

func main() {
	for _, bits := range []int{32, 64} {
		for _, order := range []string{"le", "be"} {
			path := fmt.Sprintf("examples/basic/tests/data/reloc%d%s.bin", bits, order)
			f, err := elf.Open(path)
			must(err)
			syms, err := f.Symbols()
			must(err)
			if len(syms) != 1 {
				panic("symbol count")
			}
			for _, idx := range []int{4, 5} {
				s := f.Sections[idx]
				data, err := s.Data()
				must(err)
				if s.Link != 3 || (idx == 4 && s.Info != 1) || (idx == 5 && s.Info != 0) {
					panic("section links")
				}
				for i := 0; i < len(data)/int(s.Entsize); i++ {
					reader := bytes.NewReader(data[i*int(s.Entsize):])
					var off, info uint64
					var add int64
					if bits == 32 {
						if idx == 4 {
							var r elf.Rel32
							must(binary.Read(reader, f.ByteOrder, &r))
							off, info = uint64(r.Off), uint64(r.Info)
						} else {
							var r elf.Rela32
							must(binary.Read(reader, f.ByteOrder, &r))
							off, info, add = uint64(r.Off), uint64(r.Info), int64(r.Addend)
						}
					} else {
						if idx == 4 {
							var r elf.Rel64
							must(binary.Read(reader, f.ByteOrder, &r))
							off, info = r.Off, r.Info
						} else {
							var r elf.Rela64
							must(binary.Read(reader, f.ByteOrder, &r))
							off, info, add = r.Off, r.Info, r.Addend
						}
					}
					sym, typ := uint64(elf.R_SYM64(info)), uint64(elf.R_TYPE64(info))
					max := int64(9223372036854775807)
					maxType := uint64(0xffffffff)
					if bits == 32 {
						sym, typ = uint64(elf.R_SYM32(uint32(info))), uint64(elf.R_TYPE32(uint32(info)))
						max = 2147483647
						maxType = 255
					}
					if idx == 4 {
						if i == 0 && (off != 4 || sym != 1 || typ != maxType) {
							panic("REL first")
						}
						if i == 1 && (off != 0 || sym != 0 || typ != 0) {
							panic("REL null")
						}
					} else if off != 8 || sym != 1 || typ != 2 || add != []int64{-max - 1, -1, 0, max}[i] {
						panic("RELA")
					}
				}
			}
			must(f.Close())
			fmt.Println(path, "OK")
		}
	}
}
func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
