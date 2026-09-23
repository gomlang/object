# ELF consumer

This independent module imports the versioned `ecosystem::object::elf` package. Its tests compare four generated ELF32/ELF64, little/big-endian files against Go 1.26 `debug/elf` results for header values, section and program headers, names, and symbols. `main.gom` exercises the public API.

Regenerate the reference data from `tests/data` with Go 1.26:

```sh
go run generate.go > reference.tsv
```
