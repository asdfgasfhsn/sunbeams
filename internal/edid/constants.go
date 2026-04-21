package edid

const (
	BlockSize       = 128
	DTDSize         = 18
	MaxDTDPixClkKHz = 655350 // 0xFFFF * 10 kHz

	// CVT Reduced Blanking v2
	CVTRB2HBlank = 80
	CVTRB2HFront = 8
	CVTRB2HSync  = 32
	CVTRB2HBack  = CVTRB2HBlank - CVTRB2HFront - CVTRB2HSync // 40
	CVTRB2VFront = 3
	CVTRB2VSync  = 8
	CVTRB2VBack  = 6
	CVTRB2VBlank = CVTRB2VFront + CVTRB2VSync + CVTRB2VBack // 17

	// CVT Reduced Blanking v1
	CVTRB1HBlank = 160
	CVTRB1HFront = 48
	CVTRB1HSync  = 32
	CVTRB1HBack  = CVTRB1HBlank - CVTRB1HFront - CVTRB1HSync // 80
)
