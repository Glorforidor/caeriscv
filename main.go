// This is a small simulator of the risc-v processor.
// It is going to simulate the same way as the venus simulator by kvakil
// (https://github.com/kvakil/venus).
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

type Rinst struct {
	Opcode uint8
	Rd     uint8
	Funct3 uint8
	Rs1    uint8
	Rs2    uint8
	Funct7 uint8
}

type Iinst struct {
	Opcode uint8
	Rd     uint8
	Funct3 uint8
	Rs1    uint8
	Imm    int16
}

type Sinst struct {
	Opcode uint8
	Imm1   int8
	Funct3 uint8
	Rs1    uint8
	Rs2    uint8
	Imm2   int16
}

type Uinst struct {
	Opcode uint8
	Rd     uint8
	Imm    int32
}

// testing
var (
	prog []uint32 = []uint32{
		0x00200093, // addi x1, x0, 2
		0x00300113, // addi x2, x0, 3
		0x002081b3, // add x3, x1, x2
	}
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

const usage = `Specify a binary file ending with '.bin'.`

// execute decode and executes the instruction and store the results into the
// registers.
func execute(instr uint32, reg []uint32) {
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
	case 0x33:
		rd := (instr >> 7) & 0x1f
		rs1 := (instr >> 15) & 0x1f
		rs2 := (instr >> 20) & 0x1f
		reg[rd] = reg[rs1] + reg[rs2]
	case 0x37:
		rd := (instr >> 7) & 0x1f
		imm := (instr >> 12) << 12
		reg[rd] = imm
	case 0x73:
		// Ignore Ecall for now
	default:
		fmt.Printf("Opcode %d not yet implemented\n", opcode)
	}

}

func main() {
	args := os.Args
	if len(args) < 2 || !strings.HasSuffix(args[1], ".bin") {
		fmt.Println(usage)
		os.Exit(1)
	}

	fmt.Println("Welcome to Go RISC-V simulator")
	fmt.Printf("Running program: %s\n", filepath.Base(args[1]))

	reg := make([]uint32, 32)
	prog, err := readBinary(args[1])
	if err != nil {
		panic(err)
	}
	for i, inst := range prog {
		fmt.Printf("%d: %v\n", i, inst)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintf(w, header, gen()...)

	pc := uint(0)
	for {
		instr := prog[pc]
		execute(instr, reg)
		fmt.Fprintf(w, "%v\t", pc)
		fmt.Fprintf(w, body, conv(reg)...)

		pc++
		if pc >= uint(len(prog)) {
			break
		}
	}
	w.Flush()
	fmt.Println("Program exit")
}
