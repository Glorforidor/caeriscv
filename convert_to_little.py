#!/usr/bin/env python3
import struct
import sys

if len(sys.argv) != 3:
    print(f'Usage: {sys.argv[0]} <in_file> <out_file>')
    exit(1)

in_file = open(sys.argv[1], 'rb')
out_file = open(sys.argv[2], 'wb')

try:
    data = in_file.read(4)
    while data:
        value = struct.unpack('>I', data)[0]
        new_value = struct.pack('<I', value)
        out_file.write(new_value)
        data = in_file.read(4)
finally:
    in_file.close()
    out_file.close()
