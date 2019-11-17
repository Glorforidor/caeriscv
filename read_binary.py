#!/usr/bin/env python3
import os
import struct
from struct import unpack
import sys

ending = '\n'
args = sys.argv
if len(args) < 2:
    print(f'Usage: {sys.argv[0]} <filename> [seperator]')
    print()
    print('\tDefault seperator is newline')
    exit(1)

filename = args[1]
if len(args) > 2:
    ending = args[2]

binary_file = open(filename, 'rb')

# Reading in 4 bytes at a time of the file.
iteration_count = os.fstat(binary_file.fileno()).st_size // 4
for i in range(iteration_count):
    try:
        data = binary_file.read(4)  # read the first 4 bytes
        # unpack data in lille endian (<) into integer.
        result = unpack('<I', data)[0]
    except struct.error:
        break

    print(result, end=ending)
