# `ecosystem::object::elf`

`elf::parse(bytes)` reads ELF32 and ELF64 files in either byte order. It exposes the ELF header, section and program headers, section names, static and dynamic symbols, and copied section/program data. Extended section, section-name, and program numbering are supported through section header zero; extended symbol section indexes are supported through `SHT_SYMTAB_SHNDX`.

The parser takes a byte slice and returns `Result[File, elf::Error]`. It validates table sizes and file ranges before indexing. `File` owns a copy of its input, and data accessors return separate copies. `elf::parse_with_limits(bytes, limits)` accepts caller limits. `Limits::standard()` allows at most 64 MiB input, 131,072 sections, 8,192 program headers, 1,000,000 symbols, 4,096 bytes per name, and 32 MiB per file-backed section. Valid extended program counts require a caller-supplied `max_programs` of at least 65,535. A caller reading from a file should bound the read before loading untrusted large files into memory. Names must be NUL-terminated UTF-8.

This is a bounded metadata reader. It does not resolve relocations, decompress sections, interpret notes, or construct a dynamic linker view. `SHT_NOBITS` sections have no file-backed data, so `section_data` returns empty bytes for them.

Extended symbol-index tables must link to a symbol table, use four-byte entries,
and provide exactly one entry per symbol. Duplicate companions, nonzero unused
entries, and escaped indexes outside the section table return `InvalidStructure`.
These checks follow the [ELF section-table specification](https://gabi.xinuos.com/elf/03-sheader.html).
Downstream regressions cover ELF32/ELF64 in both byte orders.

Run `(cd ../verification && just ecosystem-test object)` at the repository root for module tests, example and downstream checks, and Go `debug/elf` reference vectors.

## Development and examples

Requires GoML 0.1.56 or newer. The `examples/basic/` example shares the root manifest and its dependencies. From the library root, run:

```sh
goml run --example basic
goml test
goml verify --timeout 300s
```

`goml test` builds the example and runs its tests. `goml verify` repeats the example checks as an independent module against an isolated registry snapshot. `(cd ../verification && just ecosystem-test object)` also retains the library-specific smoke and compatibility checks.
