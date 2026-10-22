"""Independent ELF compression fixtures, using Python struct and zlib."""
from pathlib import Path
import struct
import zlib

OUT = Path(__file__).parent
PAYLOAD = bytes(range(256)) * 4
for wide in (False, True):
    for big in (False, True):
        endian = '>' if big else '<'
        header_size, section_size = (64, 64) if wide else (52, 40)
        for empty in (False, True):
            payload = b'' if empty else PAYLOAD
            compression = struct.pack(endian + ('IIQQ' if wide else 'III'), *([1, 0, len(payload), 16] if wide else [1, len(payload), 16])) + zlib.compress(payload)
            section_offset = 512
            data = bytearray(section_offset + section_size * 2)
            ident = b'\x7fELF' + bytes([2 if wide else 1, 2 if big else 1, 1]) + bytes(9)
            data[:header_size] = ident + struct.pack(endian + ('HHIQQQIHHHHHH' if wide else 'HHIIIIIHHHHHH'), 1, 62 if wide else 3, 1, 0, 0, section_offset, 0, header_size, 0, 0, section_size, 2, 0)
            data[128:128 + len(compression)] = compression
            data[section_offset + section_size:] = struct.pack(endian + ('IIQQQQIIQQ' if wide else 'IIIIIIIIII'), 0, 1, 0x800, 0, 128, len(compression), 0, 0, 8, 0)
            (OUT / f'compressed{64 if wide else 32}{"be" if big else "le"}{"_empty" if empty else ""}.bin').write_bytes(data)
