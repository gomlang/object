"""Independent struct-packed ELF note fixtures; no object-library encoder is used."""
from pathlib import Path
import struct

OUT = Path(__file__).parent
for wide in (False, True):
    for big in (False, True):
        endian = '>' if big else '<'
        for align in (4, 8):
            notes = bytearray()
            for owner, kind, desc in [(b'GNU\0', 3, b'\x01\x02\x03'), (b'', 0, b''), (b'\xff\0', 0xffffffff, b'\x0a\x00\x14\x1e\x28')]:
                notes.extend(struct.pack(endian + 'III', len(owner), len(desc), kind))
                notes.extend(owner)
                notes.extend(bytes((-len(notes)) % align))
                notes.extend(desc)
                notes.extend(bytes((-len(notes)) % align))
            header_size, program_size, section_size = (64, 56, 64) if wide else (52, 32, 40)
            data_offset, section_offset = 128, 256
            data = bytearray(section_offset + section_size * 2)
            ident = b'\x7fELF' + bytes([2 if wide else 1, 2 if big else 1, 1]) + bytes(9)
            header = struct.pack(endian + ('HHIQQQIHHHHHH' if wide else 'HHIIIIIHHHHHH'), 2, 62 if wide else 3, 1, 0, header_size, section_offset, 0, header_size, program_size, 1, section_size, 2, 0)
            data[:header_size] = ident + header
            if wide:
                program = struct.pack(endian + 'IIQQQQQQ', 4, 0, data_offset, 0, 0, len(notes), len(notes), align)
                section = struct.pack(endian + 'IIQQQQIIQQ', 0, 7, 0, 0, data_offset, len(notes), 0, 0, align, 0)
            else:
                program = struct.pack(endian + 'IIIIIIII', 4, data_offset, 0, 0, len(notes), len(notes), 0, align)
                section = struct.pack(endian + 'IIIIIIIIII', 0, 7, 0, 0, data_offset, len(notes), 0, 0, align, 0)
            data[header_size:header_size + program_size] = program
            data[data_offset:data_offset + len(notes)] = notes
            data[section_offset + section_size:] = section
            (OUT / f'notes{64 if wide else 32}{"be" if big else "le"}_{align}.bin').write_bytes(data)
