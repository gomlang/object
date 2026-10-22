# `ecosystem::object::elf`

`elf::parse(bytes)` reads ELF32 and ELF64 files in either byte order. It exposes the ELF header, section and program headers, section names, static and dynamic symbols, and copied section/program data. Extended section, section-name, and program numbering are supported through section header zero; extended symbol section indexes are supported through `SHT_SYMTAB_SHNDX`.

The parser takes a byte slice and returns `Result[File, elf::Error]`. It validates table sizes and file ranges before indexing. `File` owns a copy of its input, and data accessors return separate copies. `elf::parse_with_limits(bytes, limits)` accepts caller limits. `Limits::standard()` allows at most 64 MiB input, 131,072 sections, 8,192 program headers, 1,000,000 symbols, 4,096 bytes per name, and 32 MiB per file-backed section. Valid extended program counts require a caller-supplied `max_programs` of at least 65,535. A caller reading from a file should bound the read before loading untrusted large files into memory. Names must be NUL-terminated UTF-8.

This is a bounded metadata reader. It does not resolve relocations, interpret note descriptor payloads, or construct a dynamic linker view. Compressed section decoding is available through an explicit bounded accessor. `SHT_NOBITS` sections have no file-backed data, so `section_data` returns empty bytes for them.

Unused `PT_NULL` program headers retain their raw metadata, but their undefined
offset and size fields are ignored; `program_data` returns empty bytes. All other
program payloads must lie within the input. `PT_LOAD` additionally requires
`file_size <= memory_size` and, for alignment greater than one, a power of two
with matching virtual-address and file-offset remainders. Alignment zero or one
imposes no congruence requirement. These checks follow the
[ELF program-header specification](https://gabi.xinuos.com/elf/07-pheader.html).
They do not impose load-segment rules on notes or other supplementary metadata,
validate target page sizes or address-space limits, or constitute a complete
loader validation. Regression tests cover both classes and byte orders.

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

Run `(cd ../workflows && just ecosystem-test object)` at the repository root for module tests, example and downstream checks, and Go `debug/elf` reference vectors.

## ELF relocations

`File::section_relocations(index, RelocationLimits::standard())` reads a
`SHT_REL` or `SHT_RELA` section on demand. For an already parsed `file`, for example:

```goml
for section in file.sections() {
    if section.kind == 4 || section.kind == 9 {
        let records = file.section_relocations(section.index, elf::RelocationLimits::standard())?;
        for record in records {
            println(record.offset.to_string());
        }
    }
}
```

ELF32 and ELF64 in either byte order
use the [generic ABI relocation layout](https://gabi.xinuos.com/elf/06-reloc.html).
Each `Relocation` exposes `offset: u64`, `symbol_index: u32`, `kind: u32`,
`symbol_table_section: isize`, `target_section: Option[isize]`, and
`addend: Option[i64]`. REL has no explicit addend (`None`); RELA sign-extends
its addend. A zero `sh_info` produces `target_section: None`, as used by dynamic
relocation tables. Symbol index zero is retained; indexes refer to the original
linked symbol table, including its null entry, rather than the flattened
`symbols()`/`dynamic_symbols()` vectors (which omit null entries).

Entry sizes must match the class and record kind exactly and divide the section
size. The accessor checks symbol-table links, target-section indexes and symbol
indexes. File-backed ranges are already checked by `parse`. Relocation-specific
validation happens when calling the accessor; parsing a file alone does not
validate relocation contents. No partial result is returned on failure.
`RelocationLimits` has `max_input_bytes` and `max_relocations` fields (defaults:
32 MiB and 1,000,000 records). Negative limits fail; quotas are checked before
allocating results. Limits apply to each call, and empty tables accept zero
quotas. The existing `Limits` record is unchanged.

Offsets and unknown relocation kinds are retained without interpreting the
processor ABI. This API does not apply relocations, extract implicit REL
addends, validate relocation destinations or widths, decode RELR/Android packed
relocations, or discover relocation tables through dynamic tags without sections.
Processor-specific layouts that differ from generic ELF records are unsupported.
The independent fixtures cover both classes and byte orders, REL null-symbol
entries, and signed RELA extrema. Regenerate and cross-check them from the
library root with:

```sh
python3 elf/tests/data/make_relocations.py
go run elf/tests/data/check_relocations.go
```

## Development and examples

Requires GoML 0.1.63 or newer. Error messages use derived `ToString` templates, preserving their existing text. The `examples/basic/` example shares the root manifest and its dependencies. From the library root, run:

```sh
goml run --example basic
goml test
(cd ../workflows && just ecosystem-test object)
```

`goml test` builds the example and runs its tests. The ecosystem verifier uses an isolated registry snapshot and retains the library-specific smoke and compatibility checks. Use the source-built toolchain pinned in [workflows/ci/toolchain.json](https://github.com/gomlang/workflows/blob/main/ci/toolchain.json) for that workflow.

## Dynamic metadata and shared-library names

`File::section_dynamic(index, DynamicLimits::standard())` reads `SHT_DYNAMIC`;
`program_dynamic` reads `PT_DYNAMIC`, including files without a section table.
Both expose ordered `DynamicEntry { tag: i64, value: u64 }` records in ELF32/ELF64
and either byte order. Tags are sign-extended according to the
[ELF dynamic structure](https://gabi.xinuos.com/elf/08-dynamic.html#dynamic-section).
Unknown tags and repeated entries are retained. A `DT_NULL` terminator is required,
excluded from the result, and ends interpretation; subsequent complete entries
are ignored. The full input must still have integral entries. Section entry sizes
must be exactly 8 or 16 bytes. These are lazy accessor validations.

`section_dynamic_strings(index, tag, limits)` returns strings for `DT_NEEDED` (1),
`DT_SONAME` (14), `DT_RPATH` (15), or `DT_RUNPATH` (29), preserving order and
repeated offsets. Like [Go's debug/elf DynString](https://pkg.go.dev/debug/elf#File.DynString),
it follows the dynamic section's `sh_link` to a `SHT_STRTAB`. It requires an in-range
string offset, a terminating NUL and valid UTF-8, including for offset zero.
Other tags are rejected. For example, after locating a dynamic section:

```goml
let needed = file.section_dynamic_strings(section.index, 1, elf::DynamicLimits::standard())?;
```

The separate `DynamicLimits` record preserves existing `Limits` compatibility.
Defaults are 32 MiB of dynamic input, 1,000,000 returned entries, 4 KiB per string,
and 32 MiB total decoded string bytes per call. Negative quotas fail. The input
quota covers the entire section or segment, including ignored trailing entries;
`DT_NULL` and trailing entries do not count against `max_entries`. String budgets
exclude NUL bytes and charge repeated references separately before allocation.
A table containing only `DT_NULL` accepts a zero entry quota; zero-length input
is rejected as unterminated. No partial result is returned on failure.

This API does not resolve virtual addresses from `DT_STRTAB`, discover strings
from sectionless segments, split or expand search paths, locate libraries, or
validate mandatory dynamic-linker tags. Independent fixtures and a Go reference
check can be regenerated from the library root:

```sh
python3 elf/tests/data/make_dynamic.py
go run elf/tests/data/check_dynamic.go
```

## Compressed section data

`File::section_compression(index)` returns `Option[CompressionHeader]`, with the
raw compression `kind`, uncompressed `size`, and `alignment`; `None` denotes an
ordinary section. It reads ELF32/ELF64 compression headers in either byte order.
Unknown compression kinds remain inspectable as metadata.
`File::section_uncompressed_data(index, DecompressionLimits::standard())` returns
an owned copy of uncompressed bytes, decoding `ELFCOMPRESS_ZLIB` (1) through
`ecosystem::compress::zlib`. Ordinary sections pass through unchanged; null and
NOBITS sections return empty bytes. `section_data` still returns the raw on-disk
bytes, including compression headers, and `Section.size` remains the raw size.
This explicit split follows the approach of
[Rust object compressed data accessors](https://docs.rs/object/latest/src/object/read/elf/section.rs.html)
and provides the decoded bytes exposed by
[Go debug/elf Section.Data](https://pkg.go.dev/debug/elf#Section.Data).

The separate `DecompressionLimits` record leaves the existing `Limits` unchanged.
Its defaults permit 32 MiB of input (including the compression header), 32 MiB of
output, 1,000,000 DEFLATE blocks and 268,435,456 decoder work units per call.
All quotas must be nonnegative. Declared output size is checked before allocating
the compressed payload or decoding; the decoder also enforces that declared size
as its output ceiling. The actual output must equal the declaration. Input and
output quotas also apply to ordinary sections; blocks/work are decoder-only.
Truncated streams, checksum failures, dictionaries, trailing bytes, unsupported
codecs (including Zstd), nonzero ELF64 reserved fields, and non-power-of-two
alignment greater than one are rejected. Decoder quota exhaustion maps to
`LimitExceeded`; invalid compressed content maps to `InvalidStructure`.

Header validation is lazy and follows the
[gABI 4.3 draft section rules](https://gabi.xinuos.com/elf/03-sheader.html#section-flags):
compressed NOBITS/null sections are rejected, and SHF_ALLOC with SHF_COMPRESSED
is rejected in ET_EXEC and ET_DYN files. The draft permits this flag combination
in ET_REL, which this API accepts; Go 1.26 debug/elf rejects allocated compressed
sections regardless of file type. The independent Go oracle therefore covers
nonallocated fixtures, with separate tests for the draft's ET_REL rule.
Parsing does not automatically decompress symbol/string tables, and this API does
not recognize legacy GNU `.zdebug_*` framing or interpret DWARF.

Independent Python zlib fixtures and the Go reference check can be regenerated:

```sh
python3 elf/tests/data/make_compressed.py
go run elf/tests/data/check_compressed.go
```

## GNU symbol versions

`File::section_version_definitions(index, VersionLimits::standard())` reads
`SHT_GNU_verdef` into `VersionDefinition` records (index, flags, hash, name and
parent-version names). `section_version_needs` reads `SHT_GNU_verneed` into
`VersionNeed` records with a library name and `VersionDependency` entries
(index, hidden bit, flags, hash and name). Both use the linked string table.
The GNU record layouts have identical field widths in ELF32 and ELF64 and use
the file byte order. The APIs expose metadata comparable to
[Go debug/elf's DynamicVersions and DynamicVersionNeeds](https://pkg.go.dev/debug/elf#File.DynamicVersions).

`section_symbol_versions(index, limits)` takes a **versym section index**, follows
its `sh_link` to `SHT_DYNSYM`, and returns one `SymbolVersion` per original symbol,
**including the null symbol at index zero**. Each result contains
`symbol_table_section`, `symbol_index`, the low 15-bit version `index`, the
`hidden` bit, and optional version `name` and `library`. Definitions resolve to a
name without a library; requirements resolve to both. Index zero means local,
and one means unversioned global; both return no name/library even when verdef
contains the object's base definition at index one. `dynamic_symbols()` omits
the null entry and combines tables: first filter its symbols by `table_section`,
then use `symbol_index - 1` for nonzero indices. Existing `Symbol` is unchanged.

Resolution uses definition/requirement sections linked to the same string table
as the selected dynsym. Missing version indices greater than one, duplicate
indices, duplicate companion tables, mismatched versym/dynsym counts, and
invalid section links fail. Each standalone metadata reader also rejects
repeated indices in its own section. Format version must be one. Declared
`sh_info` and auxiliary counts must match terminating forward chains; offsets
and complete records must lie inside the section. Auxiliary lists may follow
later parent headers. Parsing does not require packed records or zero padding,
check unused bytes for overlapping interpretations, recompute version hashes,
or verify a version dependency graph. Entry sizes must be two for versym and
exactly the class's symbol-record size for its dynsym. Compressed version and
linked string/symbol sections are rejected by these accessors.

The separate `VersionLimits` preserves the existing `Limits` record. Defaults
are 32 MiB input, 1,000,000 records, 4 KiB per string and 32 MiB decoded strings.
For symbol resolution, budgets are shared across versym and all participating
version sections. Input counts complete version sections (not the already parsed
file or string tables); records count every symbol-version entry, definition,
requirement and auxiliary entry. String quotas count UTF-8 bytes excluding NUL,
charging repeated references separately before allocation. Strings must terminate
inside their table and be valid UTF-8, including offset zero. Negative limits
fail; empty version tables accept zero quotas. All checks are lazy, and a failed
call returns no partial result.

This reads section-based metadata; it does not discover sectionless version
metadata through dynamic tags or implement dynamic-linker symbol lookup rules.
Independent fixtures and a Go 1.26+ reference check cover both classes and byte
orders:

```sh
python3 elf/tests/data/make_versions.py
go run elf/tests/data/check_versions.go
```
