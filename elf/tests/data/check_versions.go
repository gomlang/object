package main

import (
	"debug/elf"
	"fmt"
	"reflect"
)

func main() {
	for _, bits := range []int{32, 64} {
		for _, order := range []string{"le", "be"} {
			path := fmt.Sprintf("elf/tests/data/versions%d%s.bin", bits, order)
			f, err := elf.Open(path)
			must(err)
			syms, err := f.DynamicSymbols()
			must(err)
			if len(syms) != 4 {
				panic("symbol count")
			}
			for i, want := range []struct {
				index            uint16
				hidden           bool
				version, library string
			}{
				{2, true, "CUSTOM_2", ""}, {3, false, "GLIBC_2.2.5", "libc.so.6"}, {1, false, "", ""}, {4, true, "WEAK_1", "libc.so.6"},
			} {
				got := syms[i]
				if !got.HasVersion || got.VersionIndex.Index() != want.index || got.VersionIndex.IsHidden() != want.hidden || got.Version != want.version || got.Library != want.library {
					panic(fmt.Sprintf("%s symbol %d: %+v", path, i, got))
				}
			}
			defs, err := f.DynamicVersions()
			must(err)
			if len(defs) != 2 || defs[0].Index != 1 || defs[0].Name != "self.so" || defs[1].Index != 2 || defs[1].Name != "CUSTOM_2" || !reflect.DeepEqual(defs[1].Deps, []string{"BASE_1"}) {
				panic(fmt.Sprintf("definitions: %+v", defs))
			}
			needs, err := f.DynamicVersionNeeds()
			must(err)
			if len(needs) != 1 || needs[0].Name != "libc.so.6" || len(needs[0].Needs) != 2 || needs[0].Needs[0].Dep != "GLIBC_2.2.5" || needs[0].Needs[1].Index != 4 {
				panic(fmt.Sprintf("needs: %+v", needs))
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
