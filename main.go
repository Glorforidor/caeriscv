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

// execute decode and executes the instruction and store the results into the
// registers. It will return whether a branch instruction is taken with an
// offset.
func execute(instr uint32, reg []uint32) (offset int, branching bool) {
	opcode := instr & 0x7f
	switch opcode {
	case 0x13:
		rd := (instr >> 7) & 0x1f
		funct3 := (instr >> 12) & 0x7
		rs1 := (instr >> 15) & 0x1f
		switch funct3 {
		case 0: // Addi
			imm := (instr >> 20)
			if imm>>11 == 1 {
				// subtract
				reg[rd] = reg[rs1] - (imm ^ 4095 + 1)
			} else {
				reg[rd] = reg[rs1] + imm
			}
		case 1: // Left shifting
			shamt := (instr >> 20) & 0x1f
			rest := (instr >> 25)
			if rest != 0 {
				fmt.Println("The encoding for left shifting is wrong:", rest)
				os.Exit(1)
			}
			v := reg[rd]
			v = v << shamt
			reg[rd] = v
		case 5:
			shamt := (instr >> 20) & 0x1f
			rest := (instr >> 25)
			if rest != 0 && rest != 32 {
				fmt.Println("The encoding for right shifting is wrong:", rest)
				os.Exit(1)
			}

			v := reg[rd]
			if rest == 0 {
				v = v >> shamt
				reg[rd] = v
			} else {
				vv := int32(v)
				vv = vv >> shamt
				reg[rd] = uint32(vv)
			}
		}
	case 0x33: // Add
		rd := (instr >> 7) & 0x1f
		rs1 := (instr >> 15) & 0x1f
		rs2 := (instr >> 20) & 0x1f
		reg[rd] = reg[rs1] + reg[rs2]
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
			offset = -2 * int((imm ^ 4095 + 1))
		} else {
			offset = 2 * int(imm)
		}

		switch funct3 {
		case 0: // BEQ
			branching = reg[rs1] == reg[rs2]
		case 1: // BNE
			branching = reg[rs1] != reg[rs2]
		case 4: // BLE
			branching = reg[rs1] < reg[rs2]
		case 5: // BGE
			branching = reg[rs1] > reg[rs2]
		}
	case 0x73: // Ecall
		fmt.Printf(body, conv(reg)...)
	default:
		fmt.Printf("Opcode %d not yet implemented\n", opcode)
	}

	return offset, branching
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

	pc := uint(0)
	for {
		instr := prog[pc]
		offset, branching := execute(instr, reg)
		if branching {
			pc = pc + uint((offset / 4))
			continue
		}

		if *debug {
			fmt.Fprintf(w, "%v\t", pc)
			fmt.Fprintf(w, body, conv(reg)...)
		}

		pc++
		if pc >= uint(len(prog)) {
			break
		}
	}
	w.Flush()
}
