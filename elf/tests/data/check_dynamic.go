package main

import (
	"debug/elf"
	"fmt"
	"reflect"
)

func main() {
	for _, bits := range []int{32, 64} {
		for _, order := range []string{"le", "be"} {
			path := fmt.Sprintf("examples/basic/tests/data/dynamic%d%s.bin", bits, order)
			f, err := elf.Open(path)
			must(err)
			// Go's DynString scans the entire section, so compare just the logical
			// array through its first DT_NULL, omitting our deliberately nonzero tail.
			f.Sections[2].Size = uint64(bits / 8 * 2 * 8)
			for tag, want := range map[elf.DynTag][]string{
				elf.DT_NEEDED: {"liba.so", "libb.so", "liba.so"},
				elf.DT_SONAME: {"self.so"}, elf.DT_RPATH: {"$ORIGIN"}, elf.DT_RUNPATH: {"$ORIGIN"},
			} {
				got, err := f.DynString(tag)
				must(err)
				if !reflect.DeepEqual(got, want) {
					panic(fmt.Sprintf("%s %s: %v", path, tag, got))
				}
			}
			unknown := elf.DynTag(-1)
			if bits == 32 {
				unknown = elf.DynTag(0xffffffff)
			}
			got, err := f.DynValue(unknown)
			must(err)
			max := ^uint64(0)
			if bits == 32 {
				max = uint64(^uint32(0))
			}
			if len(got) != 1 || got[0] != max {
				panic("unknown dynamic tag")
			}
			must(f.Close())
			fmt.Println(path, "OK")
		}
	}
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
