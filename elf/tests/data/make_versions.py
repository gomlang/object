"""Independent struct-packed GNU version fixtures (same record widths in both ELF classes)."""
from pathlib import Path
import struct
OUT = Path(__file__).parent

def elf_hash(value):
    h = 0
    for c in value.encode():
        h = (h << 4) + c
        g = h & 0xf0000000
        if g:
            h ^= g >> 24
        h &= ~g
    return h

for wide in (False, True):
    for big in (False, True):
        e = '>' if big else '<'
        h, s, width = (64, 64, 24) if wide else (52, 40, 16)
        data = bytearray(1024 + s * 7)
        ident = b'\x7fELF' + bytes([2 if wide else 1, 2 if big else 1, 1]) + bytes(9)
        data[:h] = ident + struct.pack(e + ('HHIQQQIHHHHHH' if wide else 'HHIIIIIHHHHHH'), 3, 62 if wide else 3, 1, 0, 0, 1024, 0, h, 0, 0, s, 7, 6)
        strings = bytearray(b'\0')
        offsets = {}
        for value in ['defined', 'imported', 'global', 'weak', 'self.so', 'BASE_1', 'CUSTOM_2', 'libc.so.6', 'GLIBC_2.2.5', 'WEAK_1']:
            offsets[value] = len(strings)
            strings.extend(value.encode() + b'\0')
        data[128:128+len(strings)] = strings
        symbols = bytes(width)
        for name, sec in [('defined', 4), ('imported', 0), ('global', 4), ('weak', 0)]:
            symbols += struct.pack(e + ('IBBHQQ' if wide else 'IIIBBH'), *( (offsets[name], 0x12, 0, sec, 0, 0) if wide else (offsets[name], 0, 0, 0x12, 0, sec)))
        data[256:256+len(symbols)] = symbols
        data[400:410] = struct.pack(e+'5H', 0, 0x8002, 3, 1, 0x8004)
        defs = struct.pack(e+'HHHHIII', 1, 1, 1, 1, elf_hash('self.so'), 20, 28)
        defs += struct.pack(e+'II', offsets['self.so'], 0)
        defs += struct.pack(e+'HHHHIII', 1, 0, 2, 2, elf_hash('CUSTOM_2'), 20, 0)
        defs += struct.pack(e+'II', offsets['CUSTOM_2'], 8) + struct.pack(e+'II', offsets['BASE_1'], 0)
        data[448:448+len(defs)] = defs
        needs = struct.pack(e+'HHIII', 1, 2, offsets['libc.so.6'], 16, 0)
        needs += struct.pack(e+'IHHII', elf_hash('GLIBC_2.2.5'), 0, 3, offsets['GLIBC_2.2.5'], 16)
        needs += struct.pack(e+'IHHII', elf_hash('WEAK_1'), 2, 4, offsets['WEAK_1'], 0)
        data[544:544+len(needs)] = needs
        names = b'\0.dynstr\0.dynsym\0.gnu.version\0.gnu.version_d\0.gnu.version_r\0.shstrtab\0'
        data[640:640+len(names)] = names
        def section(i, name, kind, offset, size, link=0, info=0, entry=0):
            data[1024+i*s:1024+(i+1)*s] = struct.pack(e+('IIQQQQIIQQ' if wide else 'IIIIIIIIII'), names.index(name.encode()) if name else 0, kind, 0, 0, offset, size, link, info, 1, entry)
        section(1, '.dynstr', 3, 128, len(strings))
        section(2, '.dynsym', 11, 256, len(symbols), 1, 1, width)
        section(3, '.gnu.version', 0x6fffffff, 400, 10, 2, 0, 2)
        section(4, '.gnu.version_d', 0x6ffffffd, 448, len(defs), 1, 2)
        section(5, '.gnu.version_r', 0x6ffffffe, 544, len(needs), 1, 1)
        section(6, '.shstrtab', 3, 640, len(names))
        (OUT / f'versions{64 if wide else 32}{"be" if big else "le"}.bin').write_bytes(data)

# The isolated public-consumer example owns its input; it cannot reach library tests.
consumer = OUT.parents[2] / "examples/basic/tests/data"
for fixture in OUT.glob("versions*.bin"):
    (consumer / fixture.name).write_bytes(fixture.read_bytes())
