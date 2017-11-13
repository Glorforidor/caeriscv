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

func readBinary(name string) ([]uint32, error) {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	prog := []uint32{}
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
		prog = append(prog, data)
	}
	return prog, nil
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

func gen() []interface{} {
	v := make([]interface{}, 32)
	for i := 0; i < 32; i++ {
		v[i] = i
	}
	return v
}

func main() {
	file := "shift.bin"
	fmt.Println("Welcome to Go RISC-V simulator")
	fmt.Printf("Running program: %s\n", file)

	reg := make([]uint32, 32)
	prog, err := readBinary("/home/pbj/cae-lab/finasgmt/tests/task1/" + file)
	if err != nil {
		panic(err)
	}
	for i, inst := range prog {
		fmt.Printf("%d: %v\n", i, inst)
	}

	// os.Exit(0)

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintf(w, header, gen()...)

	pc := 0
	for {
		instr := prog[pc]
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

				if rest == 0 {
					v := reg[rd]
					v = v >> shamt
					reg[rd] = v
				} else {
					v := reg[rd]
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

		fmt.Fprintf(w, "%v\t", pc)
		fmt.Fprintf(w, body, conv(reg)...)

		pc++
		if pc >= len(prog) {
			break
		}
	}
	w.Flush()
	fmt.Println("Program exit")
}
