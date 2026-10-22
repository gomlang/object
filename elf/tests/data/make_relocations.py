"""Independent gABI struct-packed REL/RELA fixtures, all classes and byte orders."""
from pathlib import Path
import struct

OUT = Path(__file__).parents[3] / 'examples/basic/tests/data'
for wide in (False, True):
    for big in (False, True):
        e = '>' if big else '<'
        h, s, sy = (64, 64, 24) if wide else (52, 40, 16)
        data = bytearray(512 + s * 6)
        ident = b'\x7fELF' + bytes([2 if wide else 1, 2 if big else 1, 1]) + bytes(9)
        data[:h] = ident + struct.pack(e + ('HHIQQQIHHHHHH' if wide else 'HHIIIIIHHHHHH'), 1, 62 if wide else 3, 1, 0, 0, 512, 0, h, 0, 0, s, 6, 0)
        def section(i, kind, offset, size, link=0, info=0, entry=0):
            data[512+i*s:512+(i+1)*s] = struct.pack(e + ('IIQQQQIIQQ' if wide else 'IIIIIIIIII'), 0, kind, 0, 0, offset, size, link, info, 1, entry)
        section(1, 1, 128, 16)
        section(2, 3, 144, 1)
        section(3, 2, 160, sy*2, 2, 1, sy)
        shift = 32 if wide else 8
        rel = b''.join(struct.pack(e + ('QQ' if wide else 'II'), off, (sym << shift) | typ) for off, sym, typ in [(4,1,0xffffffff if wide else 255),(0,0,0)])
        bits = 64 if wide else 32
        rela = b''.join(struct.pack(e + ('QQq' if wide else 'IIi'), 8, (1 << shift) | 2, addend) for addend in [-(1 << (bits-1)), -1, 0, (1 << (bits-1))-1])
        data[224:224+len(rel)] = rel
        data[288:288+len(rela)] = rela
        section(4, 9, 224, len(rel), 3, 1, 16 if wide else 8)
        section(5, 4, 288, len(rela), 3, 0, 24 if wide else 12)
        (OUT / f'reloc{bits}{"be" if big else "le"}.bin').write_bytes(data)
