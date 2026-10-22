package main

import (
	"bytes"
	"debug/elf"
	"fmt"
)

func main() {
	for _, width := range []int{32, 64} {
		for _, order := range []string{"le", "be"} {
			for _, suffix := range []string{"", "_empty"} {
				path := fmt.Sprintf("elf/tests/data/compressed%d%s%s.bin", width, order, suffix)
				f, err := elf.Open(path)
				if err != nil {
					panic(err)
				}
				got, err := f.Sections[1].Data()
				if err != nil {
					panic(err)
				}
				want := []byte{}
				if suffix == "" {
					for i := 0; i < 1024; i++ {
						want = append(want, byte(i))
					}
				}
				if !bytes.Equal(got, want) || f.Sections[1].Addralign != 16 {
					panic(path)
				}
				f.Close()
			}
		}
	}
	fmt.Println("8 compressed ELF fixtures agree with Go debug/elf")
}
