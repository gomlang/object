# `ecosystem::object::elf`

`elf::parse(bytes)` reads ELF32 and ELF64 files in either byte order. It exposes the ELF header, section and program headers, section names, static and dynamic symbols, and copied section/program data. Extended section, section-name, and program numbering are supported through section header zero; extended symbol section indexes are supported through `SHT_SYMTAB_SHNDX`.

The parser takes a byte slice and returns `Result[File, elf::Error]`. It validates table sizes and file ranges before indexing. `File` owns a copy of its input, and data accessors return separate copies. `elf::parse_with_limits(bytes, limits)` accepts caller limits. `Limits::standard()` allows at most 64 MiB input, 131,072 sections, 8,192 program headers, 1,000,000 symbols, 4,096 bytes per name, and 32 MiB per file-backed section. Valid extended program counts require a caller-supplied `max_programs` of at least 65,535. A caller reading from a file should bound the read before loading untrusted large files into memory. Names must be NUL-terminated UTF-8.

This is a bounded metadata reader. It does not resolve relocations, decompress sections, interpret note descriptor payloads, or construct a dynamic linker view. `SHT_NOBITS` sections have no file-backed data, so `section_data` returns empty bytes for them.

Extended symbol-index tables must link to a symbol table, use four-byte entries,
and provide exactly one entry per symbol. Duplicate companions, nonzero unused
entries, and escaped indexes outside the section table return `InvalidStructure`.
These checks follow the [ELF section-table specification](https://gabi.xinuos.com/elf/03-sheader.html).
Downstream regressions cover ELF32/ELF64 in both byte orders.

Decoded section and symbol names share a cumulative UTF-8 byte budget. Every
produced name is charged, including repeated references to the same string-table
entry; terminating NUL bytes and empty names are not charged. `parse` and
`parse_with_limits` use `limits.max_file_bytes` as this budget (64 MiB by default).
`parse_with_name_budget(bytes, limits, max_total_name_bytes)` selects a separate
nonnegative total while preserving the existing `Limits` record. Budget checks
precede copying and UTF-8 decoding; exhaustion returns
`LimitExceeded("total name bytes")`. Individual name limits still apply.

## ELF notes

`elf::parse_notes(bytes, endian, alignment)` decodes bounded note records;
`parse_notes_with` adds a final `NoteLimits` argument. Each `Note` exposes `name`
bytes (including the terminating NUL when nonempty), the unsigned `kind`, and
raw `descriptor` bytes. Unknown owners/types and non-UTF-8 owner bytes are retained.
A nonempty owner must end in NUL. Results own copies; padding is ignored and may
be nonzero. Empty input is valid, while incomplete headers, payloads, final
padding or trailing partial records fail without returning partial results.

`NoteAlignment::Four` and `Eight` explicitly select the record/descriptor
alignment. Both formats have three 32-bit header words and a name immediately
after the 12-byte header. The absolute descriptor offset and next record offset
are rounded up to the selected alignment. ELF class does not select the format:
GNU notes commonly use four-byte alignment in ELF64, while GNU property notes
use eight. This follows the
[elfutils note layout](https://third-party-mirror.googlesource.com/elfutils/+/4e276021fa226dfa34b803b5b1490b4e04eadd85/libelf/gelf_getnote.c).
Choose the alignment for the producer/ABI being inspected; parsing does not
try alternative formats on error.

`File::section_notes(index, alignment, limits)` and
`program_notes(index, alignment, limits)` use the file byte order and require
`SHT_NOTE`/`PT_NOTE` respectively. They apply note quotas before copying data.
`NoteLimits::standard()` permits 32 MiB input, 100,000 records, 4 KiB per owner
and 32 MiB per descriptor. Input and record caps also bound total copied bytes
and object count. The existing file `Limits` record remains unchanged.

Independent struct-packed fixtures cover ELF32/ELF64, both byte orders and both
alignments; regenerate them with `python3 elf/tests/data/make_notes.py`.

Run `(cd ../verification && just ecosystem-test object)` at the repository root for module tests, example and downstream checks, and Go `debug/elf` reference vectors.

## Development and examples

Requires GoML 0.1.56 or newer. The `examples/basic/` example shares the root manifest and its dependencies. From the library root, run:

```sh
goml run --example basic
goml test
goml verify --timeout 300s
```

`goml test` builds the example and runs its tests. `goml verify` repeats the example checks as an independent module against an isolated registry snapshot. `(cd ../verification && just ecosystem-test object)` also retains the library-specific smoke and compatibility checks.
