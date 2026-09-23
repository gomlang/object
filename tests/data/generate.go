package main

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

func align(value, boundary int) int {
	return (value + boundary - 1) / boundary * boundary
}

func sample(class int, order binary.ByteOrder) []byte {
	headerSize, programSize, sectionSize, symbolSize := 52, 32, 40, 16
	if class == 64 {
		headerSize, programSize, sectionSize, symbolSize = 64, 56, 64, 24
	}
	text := []byte{0x90, 0xc3}
	stringsData := []byte("\x00main\x00helper\x00")
	sectionNames := []byte("\x00.text\x00.symtab\x00.strtab\x00.shstrtab\x00")
	textOffset := align(headerSize+programSize, 16)
	symbolOffset := align(textOffset+len(text), 8)
	stringOffset := symbolOffset + 3*symbolSize
	nameOffset := stringOffset + len(stringsData)
	sectionOffset := align(nameOffset+len(sectionNames), 8)
	data := make([]byte, sectionOffset+5*sectionSize)
	copy(data[textOffset:], text)
	copy(data[stringOffset:], stringsData)
	copy(data[nameOffset:], sectionNames)
	copy(data[:4], []byte{0x7f, 'E', 'L', 'F'})
	if class == 64 {
		data[4] = 2
	} else {
		data[4] = 1
	}
	if order == binary.LittleEndian {
		data[5] = 1
	} else {
		data[5] = 2
	}
	data[6] = 1
	put16 := func(offset int, value uint16) { order.PutUint16(data[offset:], value) }
	put32 := func(offset int, value uint32) { order.PutUint32(data[offset:], value) }
	put64 := func(offset int, value uint64) { order.PutUint64(data[offset:], value) }
	put16(16, 2)
	if class == 64 {
		put16(18, 62)
		put32(20, 1)
		put64(24, 0x400000)
		put64(32, uint64(headerSize))
		put64(40, uint64(sectionOffset))
		put16(52, uint16(headerSize))
		put16(54, uint16(programSize))
		put16(56, 1)
		put16(58, uint16(sectionSize))
		put16(60, 5)
		put16(62, 4)
		program := headerSize
		put32(program, 1)
		put32(program+4, 5)
		put64(program+8, uint64(textOffset))
		put64(program+16, 0x400000)
		put64(program+24, 0x400000)
		put64(program+32, uint64(len(text)))
		put64(program+40, uint64(len(text)))
		put64(program+48, 16)
		for index, name := range []uint32{1, 6} {
			entry := symbolOffset + (index+1)*symbolSize
			put32(entry, name)
			data[entry+4] = 0x12
			put16(entry+6, 1)
			put64(entry+8, 0x400000+uint64(index))
			put64(entry+16, uint64(2-index))
		}
		section := func(index int, name, kind uint32, flags, address uint64, offset, size int, link, info uint32, alignment, entrySize uint64) {
			entry := sectionOffset + index*sectionSize
			put32(entry, name)
			put32(entry+4, kind)
			put64(entry+8, flags)
			put64(entry+16, address)
			put64(entry+24, uint64(offset))
			put64(entry+32, uint64(size))
			put32(entry+40, link)
			put32(entry+44, info)
			put64(entry+48, alignment)
			put64(entry+56, entrySize)
		}
		section(1, 1, 1, 6, 0x400000, textOffset, len(text), 0, 0, 16, 0)
		section(2, 7, 2, 0, 0, symbolOffset, 3*symbolSize, 3, 1, 8, uint64(symbolSize))
		section(3, 15, 3, 0, 0, stringOffset, len(stringsData), 0, 0, 1, 0)
		section(4, 23, 3, 0, 0, nameOffset, len(sectionNames), 0, 0, 1, 0)
	} else {
		put16(18, 3)
		put32(20, 1)
		put32(24, 0x400000)
		put32(28, uint32(headerSize))
		put32(32, uint32(sectionOffset))
		put16(40, uint16(headerSize))
		put16(42, uint16(programSize))
		put16(44, 1)
		put16(46, uint16(sectionSize))
		put16(48, 5)
		put16(50, 4)
		program := headerSize
		put32(program, 1)
		put32(program+4, uint32(textOffset))
		put32(program+8, 0x400000)
		put32(program+12, 0x400000)
		put32(program+16, uint32(len(text)))
		put32(program+20, uint32(len(text)))
		put32(program+24, 5)
		put32(program+28, 16)
		for index, name := range []uint32{1, 6} {
			entry := symbolOffset + (index+1)*symbolSize
			put32(entry, name)
			put32(entry+4, 0x400000+uint32(index))
			put32(entry+8, uint32(2-index))
			data[entry+12] = 0x12
			put16(entry+14, 1)
		}
		section := func(index int, name, kind, flags, address uint32, offset, size int, link, info, alignment, entrySize uint32) {
			entry := sectionOffset + index*sectionSize
			put32(entry, name)
			put32(entry+4, kind)
			put32(entry+8, flags)
			put32(entry+12, address)
			put32(entry+16, uint32(offset))
			put32(entry+20, uint32(size))
			put32(entry+24, link)
			put32(entry+28, info)
			put32(entry+32, alignment)
			put32(entry+36, entrySize)
		}
		section(1, 1, 1, 6, 0x400000, textOffset, len(text), 0, 0, 16, 0)
		section(2, 7, 2, 0, 0, symbolOffset, 3*symbolSize, 3, 1, 4, uint32(symbolSize))
		section(3, 15, 3, 0, 0, stringOffset, len(stringsData), 0, 0, 1, 0)
		section(4, 23, 3, 0, 0, nameOffset, len(sectionNames), 0, 0, 1, 0)
	}
	return data
}

func describe(name string, data []byte) {
	file, err := elf.NewFile(strings.NewReader(string(data)))
	if err != nil {
		panic(err)
	}
	defer file.Close()
	sections := make([]string, 0, len(file.Sections))
	for _, section := range file.Sections {
		sections = append(sections, section.Name)
	}
	programs := make([]string, 0, len(file.Progs))
	for _, program := range file.Progs {
		programs = append(programs, fmt.Sprint(uint32(program.Type)))
	}
	symbols, err := file.Symbols()
	if err != nil {
		panic(err)
	}
	formattedSymbols := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		formattedSymbols = append(formattedSymbols, fmt.Sprintf("%s:%d:%d:%d", symbol.Name, symbol.Value, symbol.Size, symbol.Section))
	}
	fmt.Printf("%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%s\n",
		name, file.Class, file.Data, file.Type, file.Machine, file.Entry,
		len(file.Sections), len(file.Progs), strings.Join(sections, ","),
		strings.Join(programs, ","), strings.Join(formattedSymbols, ","))
}

func main() {
	for _, class := range []int{32, 64} {
		for _, endian := range []struct {
			name  string
			order binary.ByteOrder
		}{{"le", binary.LittleEndian}, {"be", binary.BigEndian}} {
			name := fmt.Sprintf("elf%d%s.bin", class, endian.name)
			data := sample(class, endian.order)
			if err := os.WriteFile(name, data, 0644); err != nil {
				panic(err)
			}
			describe(name, data)
		}
	}
}
