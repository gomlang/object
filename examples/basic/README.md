# ELF example

This example imports the public `ecosystem::object::elf` package. Its tests compare four generated ELF32/ELF64, little/big-endian files against Go 1.26 `debug/elf` results for header values, section and program headers, names, and symbols. `main.gom` exercises the public API.

Regenerate the reference data from `tests/data` with Go 1.26:

```sh
go run generate.go > reference.tsv
```

This example shares the library root manifest and its dependencies. From the library root, run `goml verify --example basic` to build and test it as an independent downstream module.
