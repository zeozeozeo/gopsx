package emulator

import "testing"

type gteRegister struct {
	Offset uint8  // Register offset
	Value  uint32 // Register value
}

type gteConfig struct {
	Controls []gteRegister // Control registers
	Data     []gteRegister // Data registers
}

type gteTest struct {
	Desc    string    // Test description
	Initial gteConfig // Initial GTE configuration
	Command uint32    // Executed GTE command
	Result  gteConfig // GTE configuration after command
}

func TestGTE(t *testing.T) {
	for idx, test := range gteTests {
		// log test number, command, description
		t.Logf("running test %d (0x%x): %s", idx+1, test.Command, test.Desc)

		gte := test.Initial.makeGte()
		gte.Command(test.Command)
		test.Result.Validate(gte, t)
	}
}

func TestGteLZCR(t *testing.T) {
	expected := [][2]uint32{
		{0x00000000, 32},
		{0xffffffff, 32},
		{0x00000001, 31},
		{0x80000000, 1},
		{0x7fffffff, 1},
		{0xdeadbeef, 2},
		{0x000c0ffe, 12},
		{0xfffc0ffe, 14},
	}
	assert := func(v bool) {
		if !v {
			t.Error("assert failed")
		}
	}

	gte := NewGTE()
	for _, d := range expected {
		lzcs := d[0]
		lzcr := d[1]

		gte.SetData(30, lzcs)
		r := gte.Data(31)
		assert(r == lzcr)
	}
}

func (conf *gteConfig) makeGte() *GTE {
	gte := NewGTE()

	// set GTE control registers
	for _, reg := range conf.Controls {
		gte.SetControl(uint32(reg.Offset), reg.Value)
	}

	// set GTE data registers
	for _, reg := range conf.Data {
		r := reg.Offset
		v := reg.Value

		// writing to register 15 pushes a new entry onto the XY fifo
		// 28 sets the IR1...3 registers MSB
		// 29 is read only
		if r == 15 || r == 28 || r == 29 {
			continue
		}

		gte.SetData(uint32(r), v)
	}

	return gte
}

func (conf *gteConfig) Validate(gte *GTE, t *testing.T) {
	// check control registers
	for _, reg := range conf.Controls {
		v := gte.Control(uint32(reg.Offset))

		if v != reg.Value {
			t.Errorf(
				"control register %d: expected 0x%x, got 0x%x",
				reg.Offset, reg.Value, v,
			)
		}
	}

	// check data registers
	for _, reg := range conf.Data {
		v := gte.Data(uint32(reg.Offset))

		if v != reg.Value {
			t.Errorf(
				"data register %d: expected 0x%x, got 0x%x",
				reg.Offset, reg.Value, v,
			)
		}
	}
}

// Tested against the SCPH-1001 BIOS
var gteTests = []gteTest{
	// Test 1
	{
		Desc: "First GTE command used by the SCPH-1001 BIOS (RTPT)",
		Initial: gteConfig{
			Controls: []gteRegister{
				{0, 0x00000ffb},
				{1, 0xffb7ff44},
				{2, 0xf9ca0ebc},
				{3, 0x063700ad},
				{4, 0x00000eb7},
				{6, 0xfffffeac},
				{7, 0x00001700},
				{9, 0x00000fa0},
				{10, 0x0000f060},
				{11, 0x0000f060},
				{13, 0x00000640},
				{14, 0x00000640},
				{15, 0x00000640},
				{16, 0x0bb80fa0},
				{17, 0x0fa00fa0},
				{18, 0x0fa00bb8},
				{19, 0x0bb80fa0},
				{20, 0x00000fa0},
				{24, 0x01400000},
				{25, 0x00f00000},
				{26, 0x00000400},
				{27, 0xfffffec8},
				{28, 0x01400000},
				{29, 0x00000155},
				{30, 0x00000100},
			},
			Data: []gteRegister{
				{0, 0x00e70119},
				{1, 0xfffffe65},
				{2, 0x00e700d5},
				{3, 0xfffffe21},
				{4, 0x00b90119},
				{5, 0xfffffe65},
				{31, 0x00000020},
			},
		},
		Command: 0x00080030,
		Result: gteConfig{
			Controls: []gteRegister{
				{0, 0x00000ffb},
				{1, 0xffb7ff44},
				{2, 0xf9ca0ebc},
				{3, 0x063700ad},
				{4, 0x00000eb7},
				{6, 0xfffffeac},
				{7, 0x00001700},
				{9, 0x00000fa0},
				{10, 0x0000f060},
				{11, 0x0000f060},
				{13, 0x00000640},
				{14, 0x00000640},
				{15, 0x00000640},
				{16, 0x0bb80fa0},
				{17, 0x0fa00fa0},
				{18, 0x0fa00bb8},
				{19, 0x0bb80fa0},
				{20, 0x00000fa0},
				{24, 0x01400000},
				{25, 0x00f00000},
				{26, 0x00000400},
				{27, 0xfffffec8},
				{28, 0x01400000},
				{29, 0x00000155},
				{30, 0x00000100},
				{31, 0x00001000},
			},
			Data: []gteRegister{
				{0, 0x00e70119},
				{1, 0xfffffe65},
				{2, 0x00e700d5},
				{3, 0xfffffe21},
				{4, 0x00b90119},
				{5, 0xfffffe65},
				{8, 0x00001000},
				{9, 0x0000012b},
				{10, 0xfffffff0},
				{11, 0x000015d9},
				{12, 0x00f40176},
				{13, 0x00f9016b},
				{14, 0x00ed0176},
				{15, 0x00ed0176},
				{17, 0x000015eb},
				{18, 0x000015aa},
				{19, 0x000015d9},
				{24, 0x0106e038},
				{25, 0x0000012b},
				{26, 0xfffffff0},
				{27, 0x000015d9},
				{28, 0x00007c02},
				{29, 0x00007c02},
				{31, 0x00000020},
			},
		},
	},
	// Test 2
	{
		Desc: "2nd GTE command: RTPT",
		Initial: gteConfig{
			Controls: []gteRegister{
				{0, 0x00000ffb},
				{1, 0xffb7ff44},
				{2, 0xf9ca0ebc},
				{3, 0x063700ad},
				{4, 0x00000eb7},
				{6, 0xfffffeac},
				{7, 0x00001700},
				{9, 0x00000fa0},
				{10, 0x0000f060},
				{11, 0x0000f060},
				{13, 0x00000640},
				{14, 0x00000640},
				{15, 0x00000640},
				{16, 0x0bb80fa0},
				{17, 0x0fa00fa0},
				{18, 0x0fa00bb8},
				{19, 0x0bb80fa0},
				{20, 0x00000fa0},
				{24, 0x01400000},
				{25, 0x00f00000},
				{26, 0x00000400},
				{27, 0xfffffec8},
				{28, 0x01400000},
				{29, 0x00000155},
				{30, 0x00000100},
				{31, 0x00001000},
			},
			Data: []gteRegister{
				{0, 0x00e70119},
				{1, 0xfffffe65},
				{2, 0x00e700d5},
				{3, 0xfffffe21},
				{4, 0x00b90119},
				{5, 0xfffffe65},
				{8, 0x00001000},
				{9, 0x0000012b},
				{10, 0xfffffff0},
				{11, 0x000015d9},
				{12, 0x00f40176},
				{13, 0x00f9016b},
				{14, 0x00ed0176},
				{15, 0x00ed0176},
				{17, 0x000015eb},
				{18, 0x000015aa},
				{19, 0x000015d9},
				{24, 0x0106e038},
				{25, 0x0000012b},
				{26, 0xfffffff0},
				{27, 0x000015d9},
				{28, 0x00007c02},
				{29, 0x00007c02},
				{31, 0x00000020},
			},
		},
		Command: 0x00000006,
		Result: gteConfig{
			Controls: []gteRegister{
				{0, 0x00000ffb},
				{1, 0xffb7ff44},
				{2, 0xf9ca0ebc},
				{3, 0x063700ad},
				{4, 0x00000eb7},
				{6, 0xfffffeac},
				{7, 0x00001700},
				{9, 0x00000fa0},
				{10, 0x0000f060},
				{11, 0x0000f060},
				{13, 0x00000640},
				{14, 0x00000640},
				{15, 0x00000640},
				{16, 0x0bb80fa0},
				{17, 0x0fa00fa0},
				{18, 0x0fa00bb8},
				{19, 0x0bb80fa0},
				{20, 0x00000fa0},
				{24, 0x01400000},
				{25, 0x00f00000},
				{26, 0x00000400},
				{27, 0xfffffec8},
				{28, 0x01400000},
				{29, 0x00000155},
				{30, 0x00000100},
			},
			Data: []gteRegister{
				{0, 0x00e70119},
				{1, 0xfffffe65},
				{2, 0x00e700d5},
				{3, 0xfffffe21},
				{4, 0x00b90119},
				{5, 0xfffffe65},
				{8, 0x00001000},
				{9, 0x0000012b},
				{10, 0xfffffff0},
				{11, 0x000015d9},
				{12, 0x00f40176},
				{13, 0x00f9016b},
				{14, 0x00ed0176},
				{15, 0x00ed0176},
				{17, 0x000015eb},
				{18, 0x000015aa},
				{19, 0x000015d9},
				{24, 0x0000004d},
				{25, 0x0000012b},
				{26, 0xfffffff0},
				{27, 0x000015d9},
				{28, 0x00007c02},
				{29, 0x00007c02},
				{31, 0x00000020},
			},
		},
	},
	// Test 3
	{
		Desc: "3nd GTE command: AVSZ3",
		Initial: gteConfig{
			Controls: []gteRegister{
				{0, 0x00000ffb},
				{1, 0xffb7ff44},
				{2, 0xf9ca0ebc},
				{3, 0x063700ad},
				{4, 0x00000eb7},
				{6, 0xfffffeac},
				{7, 0x00001700},
				{9, 0x00000fa0},
				{10, 0x0000f060},
				{11, 0x0000f060},
				{13, 0x00000640},
				{14, 0x00000640},
				{15, 0x00000640},
				{16, 0x0bb80fa0},
				{17, 0x0fa00fa0},
				{18, 0x0fa00bb8},
				{19, 0x0bb80fa0},
				{20, 0x00000fa0},
				{24, 0x01400000},
				{25, 0x00f00000},
				{26, 0x00000400},
				{27, 0xfffffec8},
				{28, 0x01400000},
				{29, 0x00000155},
				{30, 0x00000100},
			},
			Data: []gteRegister{
				{0, 0x00e70119},
				{1, 0xfffffe65},
				{2, 0x00e700d5},
				{3, 0xfffffe21},
				{4, 0x00b90119},
				{5, 0xfffffe65},
				{8, 0x00001000},
				{9, 0x0000012b},
				{10, 0xfffffff0},
				{11, 0x000015d9},
				{12, 0x00f40176},
				{13, 0x00f9016b},
				{14, 0x00ed0176},
				{15, 0x00ed0176},
				{17, 0x000015eb},
				{18, 0x000015aa},
				{19, 0x000015d9},
				{24, 0x0000004d},
				{25, 0x0000012b},
				{26, 0xfffffff0},
				{27, 0x000015d9},
				{28, 0x00007c02},
				{29, 0x00007c02},
				{31, 0x00000020},
			},
		},
		Command: 0x0008002d,
		Result: gteConfig{
			Controls: []gteRegister{
				{0, 0x00000ffb},
				{1, 0xffb7ff44},
				{2, 0xf9ca0ebc},
				{3, 0x063700ad},
				{4, 0x00000eb7},
				{6, 0xfffffeac},
				{7, 0x00001700},
				{9, 0x00000fa0},
				{10, 0x0000f060},
				{11, 0x0000f060},
				{13, 0x00000640},
				{14, 0x00000640},
				{15, 0x00000640},
				{16, 0x0bb80fa0},
				{17, 0x0fa00fa0},
				{18, 0x0fa00bb8},
				{19, 0x0bb80fa0},
				{20, 0x00000fa0},
				{24, 0x01400000},
				{25, 0x00f00000},
				{26, 0x00000400},
				{27, 0xfffffec8},
				{28, 0x01400000},
				{29, 0x00000155},
				{30, 0x00000100},
			},
			Data: []gteRegister{
				{0, 0x00e70119},
				{1, 0xfffffe65},
				{2, 0x00e700d5},
				{3, 0xfffffe21},
				{4, 0x00b90119},
				{5, 0xfffffe65},
				{7, 0x00000572},
				{8, 0x00001000},
				{9, 0x0000012b},
				{10, 0xfffffff0},
				{11, 0x000015d9},
				{12, 0x00f40176},
				{13, 0x00f9016b},
				{14, 0x00ed0176},
				{15, 0x00ed0176},
				{17, 0x000015eb},
				{18, 0x000015aa},
				{19, 0x000015d9},
				{24, 0x00572786},
				{25, 0x0000012b},
				{26, 0xfffffff0},
				{27, 0x000015d9},
				{28, 0x00007c02},
				{29, 0x00007c02},
				{31, 0x00000020},
			},
		},
	},
	// Test 4
	{
		Desc: "First NCDS command",
		Initial: gteConfig{
			Controls: []gteRegister{
				{0, 0x00000ffb},
				{1, 0xffb7ff44},
				{2, 0xf9ca0ebc},
				{3, 0x063700ad},
				{4, 0x00000eb7},
				{6, 0xfffffeac},
				{7, 0x00001700},
				{9, 0x00000fa0},
				{10, 0x0000f060},
				{11, 0x0000f060},
				{13, 0x00000640},
				{14, 0x00000640},
				{15, 0x00000640},
				{16, 0x0bb80fa0},
				{17, 0x0fa00fa0},
				{18, 0x0fa00bb8},
				{19, 0x0bb80fa0},
				{20, 0x00000fa0},
				{24, 0x01400000},
				{25, 0x00f00000},
				{26, 0x00000400},
				{27, 0xfffffec8},
				{28, 0x01400000},
				{29, 0x00000155},
				{30, 0x00000100},
			},
			Data: []gteRegister{
				{0, 0x00000b50},
				{1, 0xfffff4b0},
				{2, 0x00e700d5},
				{3, 0xfffffe21},
				{4, 0x00b90119},
				{5, 0xfffffe65},
				{6, 0x2094a539},
				{7, 0x00000572},
				{8, 0x00001000},
				{9, 0x0000012b},
				{10, 0xfffffff0},
				{11, 0x000015d9},
				{12, 0x00f40176},
				{13, 0x00f9016b},
				{14, 0x00ed0176},
				{15, 0x00ed0176},
				{17, 0x000015eb},
				{18, 0x000015aa},
				{19, 0x000015d9},
				{24, 0x00572786},
				{25, 0x0000012b},
				{26, 0xfffffff0},
				{27, 0x000015d9},
				{28, 0x00007c02},
				{29, 0x00007c02},
				{31, 0x00000020},
			},
		},
		Command: 0x00080413,
		Result: gteConfig{
			Controls: []gteRegister{
				{0, 0x00000ffb},
				{1, 0xffb7ff44},
				{2, 0xf9ca0ebc},
				{3, 0x063700ad},
				{4, 0x00000eb7},
				{6, 0xfffffeac},
				{7, 0x00001700},
				{9, 0x00000fa0},
				{10, 0x0000f060},
				{11, 0x0000f060},
				{13, 0x00000640},
				{14, 0x00000640},
				{15, 0x00000640},
				{16, 0x0bb80fa0},
				{17, 0x0fa00fa0},
				{18, 0x0fa00bb8},
				{19, 0x0bb80fa0},
				{20, 0x00000fa0},
				{24, 0x01400000},
				{25, 0x00f00000},
				{26, 0x00000400},
				{27, 0xfffffec8},
				{28, 0x01400000},
				{29, 0x00000155},
				{30, 0x00000100},
				{31, 0x81f00000},
			},
			Data: []gteRegister{
				{0, 0x00000b50},
				{1, 0xfffff4b0},
				{2, 0x00e700d5},
				{3, 0xfffffe21},
				{4, 0x00b90119},
				{5, 0xfffffe65},
				{6, 0x2094a539},
				{7, 0x00000572},
				{8, 0x00001000},
				{12, 0x00f40176},
				{13, 0x00f9016b},
				{14, 0x00ed0176},
				{15, 0x00ed0176},
				{17, 0x000015eb},
				{18, 0x000015aa},
				{19, 0x000015d9},
				{22, 0x20000000},
				{24, 0x00572786},
				{25, 0xffffffff},
				{26, 0xffffffff},
				{31, 0x00000020},
			},
		},
	},
}
