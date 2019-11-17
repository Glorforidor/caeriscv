// This is a small simulator of the risc-v processor.
// It is going to simulate the same way as the venus simulator by kvakil
// (https://github.com/kvakil/venus).
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

// readBinary reads binary file in little endian format and returns the content
// in slice of instructions. If there is an error, it will be of type ErrUnexpectedEOF.
func readBinary(name string) (instructions []uint32, err error) {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	var data uint32
	buf := bytes.NewReader(b)
	for {
		err = binary.Read(buf, binary.LittleEndian, &data)
		if err != nil {
			if err != io.EOF {
				return nil, fmt.Errorf("could not decode binary data: %v", err)
			}
			break
		}
		instructions = append(instructions, data)
	}
	return instructions, nil
}

// writeBinary writes registers out in binary format to named file.
// If there is an error, it is either a file creation or binary writing failure.
func writeBinary(name string, reg []uint32) error {
	f, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("could not create file: %v", err)
	}
	defer f.Close()

	err = binary.Write(f, binary.LittleEndian, reg)
	if err != nil {
		return fmt.Errorf("could not write content in binary: %v", err)
	}

	return nil
}

const header = "PC\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\tx%v\t\n"
const body = "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t\n"

func conv(a []uint32) []interface{} {
	s := make([]interface{}, len(a))
	for i, v := range a {
		s[i] = v
	}
	return s
}

// gen generates a slice from 0 to 31.
func gen() []interface{} {
	v := make([]interface{}, 32)
	for i := 0; i < 32; i++ {
		v[i] = i
	}
	return v
}

// sext sign extend a imm value.
func sext(imm uint32) uint32 {
	if imm>>20 == 1 {
		imm = imm | 0xfff00000
	} else if imm>>11 == 1 {
		imm = imm | 0xfffff000
	}
	return imm
}

// execute decode and executes the instruction and store the results into the
// registers. It will return whether a branch instruction is taken with an
// offset.
func execute(pc, instr uint32, reg []uint32, mem []byte) (offset int, branching, exit bool) {
	opcode := instr & 0x7f
	switch opcode {
	case 0x3:
		rd := (instr >> 7) & 0x1f
		funct3 := (instr >> 12) & 0x7
		rs1 := (instr >> 15) & 0x1f
		imm := sext((instr >> 20))
		sp := reg[rs1]
		switch funct3 {
		case 0: // LB
			reg[rd] = uint32(int8(mem[sp+imm]))
		case 1: // LH
			res := uint32(0)
			for i := 0; i < 2; i++ {
				res = res + uint32(int16(mem[sp+imm+uint32(i)])<<uint(8*i))
			}
			reg[rd] = res
		case 2: // LW
			res := uint32(0)
			for i := 0; i < 4; i++ {
				res = res + uint32(int32(mem[sp+imm+uint32(i)])<<uint(8*i))
			}
			reg[rd] = res
		case 4: // LBU
			reg[rd] = uint32(mem[sp+imm])
		case 5: // LHU
			res := uint32(0)
			for i := 0; i < 2; i++ {
				res = res + uint32(uint16(mem[sp+imm+uint32(i)])<<uint(8*i))
			}
			reg[rd] = res
		}
	case 0x13:
		rd := (instr >> 7) & 0x1f
		funct3 := (instr >> 12) & 0x7
		rs1 := (instr >> 15) & 0x1f
		imm := sext((instr >> 20))
		switch funct3 {
		case 0: // Addi
			reg[rd] = reg[rs1] + imm
		case 1: // Shift Left Logical Intermediate
			shamt := imm & 0x3f
			rest := (imm >> 6)
			if rest == 0 {
				reg[rd] = reg[rs1] << shamt
			}
		case 2: // SLTI
			trs1 := int32(reg[rs1])
			timm := int32(imm)
			if trs1 < timm {
				reg[rd] = 1
			} else {
				reg[rd] = 0
			}
		case 3: // SLTIU
			if reg[rs1] < imm {
				reg[rd] = 1
			} else {
				reg[rd] = 0
			}
		case 4: // XOR Intermediate
			reg[rd] = reg[rs1] ^ imm
		case 5: // Shift Right Intermediate
			shamt := imm & 0x3f
			rest := (imm >> 6)

			if rest == 0 { // Logical
				reg[rd] = reg[rs1] >> shamt
			} else { // Arithmetic
				reg[rd] = uint32(int32(reg[rs1]) >> shamt)
			}
		case 6: // OR Intermediate
			reg[rd] = reg[rs1] | imm
		case 7: // AND Intermediate
			reg[rd] = reg[rs1] & imm
		}
	case 0x17: // AUIPC
		rd := (instr >> 7) & 0x1f
		imm := (instr >> 12) << 12
		reg[rd] = pc + imm
	case 0x23:
		imm1 := (instr >> 7) & 0x1f
		funct3 := (instr >> 12) & 0x7
		rs1 := (instr >> 15) & 0x1f // base
		rs2 := (instr >> 20) & 0x1f // src
		imm2 := (instr >> 25)
		imm := sext(imm2<<5 + imm1)
		sp := reg[rs1]
		switch funct3 {
		case 0: // SB
			mem[sp+imm] = byte(reg[rs2] & 0xff)
		case 1: // SH
			for i := 0; i < 2; i++ {
				mem[sp+imm+uint32(i)] = byte((uint16(reg[rs2]) >> uint(8*i)) & 0xff)
			}
		case 2: // SW
			for i := 0; i < 4; i++ {
				mem[sp+imm+uint32(i)] = byte((uint32(reg[rs2]) >> uint(8*i)) & 0xff)
			}
		}
	case 0x33:
		rd := (instr >> 7) & 0x1f
		funct3 := (instr >> 12) & 0x7
		rs1 := (instr >> 15) & 0x1f
		rs2 := (instr >> 20) & 0x1f
		funct7 := (instr >> 25)
		switch funct3 {
		case 0:
			switch funct7 {
			case 0: // Add
				reg[rd] = reg[rs1] + reg[rs2]
			case 1: // Mul
				reg[rd] = reg[rs1] * reg[rs2]
			case 32: // Sub
				reg[rd] = reg[rs1] - reg[rs2]
			}
		case 1:
			switch funct7 {
			case 0: // Shift Left Logical
				reg[rd] = reg[rs1] << reg[rs2]
			case 1: // Mulh
				res := int64(int32(reg[rs1])) * int64(int32(reg[rs2]))
				res = res >> 32
				reg[rd] = uint32(res)
			}
		case 2:
			switch funct7 {
			case 0: // SLT
				trs1 := int32(reg[rs1])
				trs2 := int32(reg[rs2])
				if trs1 < trs2 {
					reg[rd] = 1
				} else {
					reg[rd] = 0
				}
			case 1: // Mulhsu
				res := int64(int32(reg[rs1])) * int64(reg[rs2])
				res = res >> 32
				reg[rd] = uint32(res)
			}
		case 3:
			switch funct7 {
			case 0: // SLTU
				if reg[rs1] < reg[rs2] {
					reg[rd] = 1
				} else {
					reg[rd] = 0
				}
			case 1: // Mulhu
				res := uint64(reg[rs1]) * uint64(reg[rs2])
				res = res >> 32
				reg[rd] = uint32(res)
			}
		case 4:
			switch funct7 {
			case 0: // XOR
				reg[rd] = reg[rs1] ^ reg[rs2]
			case 1: // Div
				if int32(reg[rs2]) == 0 {
					reg[rd] = ^uint32(0)
				} else {
					reg[rd] = uint32(int32(reg[rs1]) / int32(reg[rs2]))
				}
			}
		case 5: // Shift Right
			switch funct7 {
			case 0: // Logical
				reg[rd] = reg[rs1] >> reg[rs2]
			case 1: // Divu
				// TODO: ask TA about unsigned division by zero.
				if reg[rs2] == 0 {
					reg[rd] = reg[rs1]
				} else {
					reg[rd] = reg[rs1] / reg[rs2]
				}
			case 32: // Arithmetic
				reg[rd] = uint32(int32(reg[rs1]) >> reg[rs2])
			}
		case 6:
			switch funct7 {
			case 0: // OR
				reg[rd] = reg[rs1] | reg[rs2]
			case 1: // Rem
				if reg[rs2] == 0 {
					reg[rd] = uint32(int32(reg[rs1]))
				} else {
					reg[rd] = uint32(int32(reg[rs1]) % int32(reg[rs2]))
				}
			}
		case 7:
			switch funct7 {
			case 0: // AND
				reg[rd] = reg[rs1] & reg[rs2]
			case 1: // Remu
				if reg[rs2] == 0 {
					reg[rd] = reg[rs1]
				} else {
					reg[rd] = reg[rs1] % reg[rs2]
				}
			}
		}
	case 0x37: // LUI
		rd := (instr >> 7) & 0x1f
		imm := (instr >> 12) << 12
		reg[rd] = imm
	case 0x63: // Branching
		imm1 := (instr >> 7) & 0x1 // imm 11
		imm2 := (instr >> 8) & 0xf // imm 1 - 4
		funct3 := (instr >> 12) & 0x7
		rs1 := (instr >> 15) & 0x1f
		rs2 := (instr >> 20) & 0x1f
		imm3 := (instr >> 25) & 0x3f // imm 5 - 10
		imm4 := (instr >> 31)        // imm 12
		imm := imm4<<11 + imm1<<10 + imm3<<4 + imm2

		if imm4 == 1 {
			offset = -2 * int(imm^0xfff+1)
		} else {
			offset = 2 * int(imm)
		}

		switch funct3 {
		case 0: // BEQ
			branching = reg[rs1] == reg[rs2]
		case 1: // BNE
			branching = reg[rs1] != reg[rs2]
		case 4: // BLT
			branching = int32(reg[rs1]) < int32(reg[rs2])
		case 5: // BGE
			branching = int32(reg[rs1]) >= int32(reg[rs2])
		case 6: // BLTU
			branching = reg[rs1] < reg[rs2]
		case 7: // BGEU
			branching = reg[rs1] >= reg[rs2]
		}
	case 0x67: // JALR
		rd := (instr >> 7) & 0x1f
		funct3 := (instr >> 12) & 0x7
		rs1 := (instr >> 15) & 0x1f
		imm := sext((instr >> 20))
		if funct3 == 0 {
			branching = true
			reg[rd] = pc + 1
			offset = int(reg[rs1]+imm) & 0xfffffffe
		}
	case 0x6f: // JAL
		rd := (instr >> 7) & 0x1f
		imm1 := (instr >> 12) & 0xff  // imm 12 - 19
		imm2 := (instr >> 20) & 0x1   // imm 11
		imm3 := (instr >> 21) & 0x3ff // imm 1 - 10
		imm4 := (instr >> 31)         // imm 20
		imm := sext((imm4<<20 + imm1<<12 + imm2<<11 + imm3<<1))
		branching = true
		reg[rd] = pc + 1
		offset = int(reg[rd] + imm)
	case 0x73: // Ecall
		fmt.Println(conv(reg)...)
		exit = true
	default:
		fmt.Printf("Opcode %d not yet implemented\n", opcode)
	}

	reg[0] = 0

	return offset, branching, exit
}

func usage() {
	fmt.Println(`Usage: caeriscv [-debug] <binary file>`)
	flag.PrintDefaults()
}

func main() {
	debug := flag.Bool("debug", false, "enable debug information")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 || !strings.HasSuffix(args[0], ".bin") {
		usage()
		os.Exit(1)
	}

	reg := make([]uint32, 32)
	mem := make([]byte, 4096)
	reg[2] = uint32(len(mem))
	prog, err := readBinary(args[0])
	if err != nil {
		panic(err)
	}

	w := new(tabwriter.Writer)
	if *debug {
		w.Init(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
		fmt.Fprintln(w, "Welcome to Go RISC-V simulator")
		fmt.Fprintf(w, "Running program: %s\n", filepath.Base(args[0]))
		fmt.Fprintln(w, "Instructions:")
		for i, instr := range prog {
			fmt.Fprintf(w, "%d: %v\n", i, instr)
		}
		fmt.Fprintln(w)
		fmt.Fprintf(w, header, gen()...)
	}

	pc := uint32(0)
	for {
		instr := prog[pc]
		offset, branching, exit := execute(pc, instr, reg, mem)
		if *debug {
			fmt.Fprintf(w, "%v\t", pc)
			fmt.Fprintf(w, body, conv(reg)...)
		}
		if exit {
			break
		}
		if branching {
			pc = pc + uint32((offset / 4))
			continue
		}

		pc++
		if pc >= uint32(len(prog)) {
			break
		}
	}
	w.Flush()
	err = writeBinary("out.res", reg)
	if err != nil {
		panic(err)
	}
}
