"""Independent gABI dynamic fixtures for both ELF classes and byte orders."""
from pathlib import Path
import struct

OUT = Path(__file__).parents[3] / 'examples/basic/tests/data'
for wide in (False, True):
    for big in (False, True):
        e = '>' if big else '<'
        h, p, s, width = (64, 56, 64, 8) if wide else (52, 32, 40, 4)
        data = bytearray(512 + s * 3)
        ident = b'\x7fELF' + bytes([2 if wide else 1, 2 if big else 1, 1]) + bytes(9)
        data[:h] = ident + struct.pack(e + ('HHIQQQIHHHHHH' if wide else 'HHIIIIIHHHHHH'), 3, 62 if wide else 3, 1, 0, h, 512, 0, h, p, 1, s, 3, 0)
        strings = b'\0liba.so\0libb.so\0self.so\0$ORIGIN\0'
        data[384:384+len(strings)] = strings
        entries = [(1,1),(1,9),(1,1),(14,17),(15,25),(29,25),(-1,(1 << (width*8))-1),(0,99),(1,9)]
        dynamic = b''.join(struct.pack(e + ('qQ' if wide else 'iI'), *v) for v in entries)
        data[160:160+len(dynamic)] = dynamic
        prog = (2, 4, 160, 0, 0, len(dynamic), len(dynamic), width) if wide else (2,160,0,0,len(dynamic),len(dynamic),4,width)
        data[h:h+p] = struct.pack(e + ('IIQQQQQQ' if wide else 'IIIIIIII'), *prog)
        def section(i, kind, offset, size, link=0, entry=0):
            data[512+i*s:512+(i+1)*s] = struct.pack(e + ('IIQQQQIIQQ' if wide else 'IIIIIIIIII'), 0, kind, 0, 0, offset, size, link, 0, 1, entry)
        section(1,3,384,len(strings))
        section(2,6,160,len(dynamic),1,width*2)
        (OUT / f'dynamic{width*8}{"be" if big else "le"}.bin').write_bytes(data)
