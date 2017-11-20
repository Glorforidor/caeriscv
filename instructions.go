package main

// TODO: maybe use these structs in the future.
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
