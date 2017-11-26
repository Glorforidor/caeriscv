package main

// TODO: maybe use these structs in the future.
type rinst struct {
	Opcode uint8
	Rd     uint8
	Funct3 uint8
	Rs1    uint8
	Rs2    uint8
	Funct7 uint8
}

type iinst struct {
	Opcode uint8
	Rd     uint8
	Funct3 uint8
	Rs1    uint8
	Imm    int16
}

type sinst struct {
	Opcode uint8
	Imm1   int8
	Funct3 uint8
	Rs1    uint8
	Rs2    uint8
	Imm2   int16
}

type uinst struct {
	Opcode uint8
	Rd     uint8
	Imm    int32
}
