package main

import "testing"

func TestBit32Arshift(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			"sign extension basic",
			`local r = bit32.arshift(0x80000000, 1)
			assert(r == 0xC0000000, "expected 0xC0000000, got " .. string.format("0x%X", r))`,
		},
		{
			"sign extension fills all bits",
			`local r = bit32.arshift(0x80000000, 31)
			assert(r == 0xFFFFFFFF, "expected 0xFFFFFFFF, got " .. string.format("0x%X", r))`,
		},
		{
			"sign extension shift >= 32",
			`local r = bit32.arshift(0x80000000, 32)
			assert(r == 0xFFFFFFFF, "expected 0xFFFFFFFF for shift >= 32 with sign bit")`,
		},
		{
			"no sign extension when MSB clear",
			`local r = bit32.arshift(0x40000000, 1)
			assert(r == 0x20000000, "expected 0x20000000, got " .. string.format("0x%X", r))`,
		},
		{
			"no sign extension shift >= 32",
			`local r = bit32.arshift(0x7FFFFFFF, 32)
			assert(r == 0, "expected 0 for shift >= 32 with clear sign bit")`,
		},
		{
			"arshift by 0",
			`assert(bit32.arshift(0xDEADBEEF, 0) == 0xDEADBEEF)`,
		},
		{
			"arshift negative disp is lshift",
			`assert(bit32.arshift(1, -4) == 16)`,
		},
		{
			"arshift 0xFF000000 by 8",
			`local r = bit32.arshift(0xFF000000, 8)
			assert(r == 0xFFFF0000, "expected 0xFFFF0000, got " .. string.format("0x%X", r))`,
		},
		{
			"all ones stays all ones",
			`assert(bit32.arshift(0xFFFFFFFF, 1) == 0xFFFFFFFF)
			assert(bit32.arshift(0xFFFFFFFF, 16) == 0xFFFFFFFF)
			assert(bit32.arshift(0xFFFFFFFF, 31) == 0xFFFFFFFF)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLuaSource(t, tt.source, tt.name)
		})
	}
}

func TestBit32ShiftEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			"lshift by 32 returns 0",
			`assert(bit32.lshift(0xFFFFFFFF, 32) == 0)`,
		},
		{
			"rshift by 32 returns 0",
			`assert(bit32.rshift(0xFFFFFFFF, 32) == 0)`,
		},
		{
			"lshift large displacement",
			`assert(bit32.lshift(1, 100) == 0)`,
		},
		{
			"rshift large displacement",
			`assert(bit32.rshift(1, 100) == 0)`,
		},
		{
			"lshift negative is rshift",
			`assert(bit32.lshift(0x100, -4) == 0x10)`,
		},
		{
			"rshift negative is lshift",
			`assert(bit32.rshift(0x10, -4) == 0x100)`,
		},
		{
			"rshift does not sign extend",
			`local r = bit32.rshift(0x80000000, 1)
			assert(r == 0x40000000, "rshift must be logical, got " .. string.format("0x%X", r))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLuaSource(t, tt.source, tt.name)
		})
	}
}

func TestBit32InputTruncation(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			"large input truncated to uint32",
			`assert(bit32.band(0x1FFFFFFFF, 0xFFFFFFFF) == 0xFFFFFFFF)`,
		},
		{
			"negative input wraps as uint32",
			`-- -1 as uint32 = 0xFFFFFFFF
			assert(bit32.band(-1, 0xFF) == 0xFF)`,
		},
		{
			"bnot of truncated value",
			`assert(bit32.bnot(-1) == 0, "bnot(-1) should be 0 since -1 truncates to 0xFFFFFFFF")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLuaSource(t, tt.source, tt.name)
		})
	}
}

func TestBit32ExtractReplace(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			"extract nibble",
			`assert(bit32.extract(0xAC, 4, 4) == 10)`,
		},
		{
			"extract single bit default width",
			`assert(bit32.extract(5, 0) == 1)
			assert(bit32.extract(5, 1) == 0)
			assert(bit32.extract(5, 2) == 1)`,
		},
		{
			"extract full 32 bits",
			`assert(bit32.extract(0xDEADBEEF, 0, 32) == 0xDEADBEEF)`,
		},
		{
			"replace low nibble",
			`assert(bit32.replace(0xF0, 0x0A, 0, 4) == 0xFA)`,
		},
		{
			"replace single bit",
			`assert(bit32.replace(0, 1, 7) == 128)`,
		},
		{
			"extract then replace roundtrip",
			`local orig = 0xDEADBEEF
			local hi = bit32.extract(orig, 16, 16)
			local lo = bit32.extract(orig, 0, 16)
			local rebuilt = bit32.replace(0, lo, 0, 16)
			rebuilt = bit32.replace(rebuilt, hi, 16, 16)
			assert(rebuilt == orig, "roundtrip failed")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLuaSource(t, tt.source, tt.name)
		})
	}
}

func TestBit32Rotate(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			"lrotate wraps MSB to LSB",
			`assert(bit32.lrotate(0x80000000, 1) == 1)`,
		},
		{
			"rrotate wraps LSB to MSB",
			`assert(bit32.rrotate(1, 1) == 0x80000000)`,
		},
		{
			"full rotation is identity",
			`assert(bit32.lrotate(0x12345678, 32) == 0x12345678)
			assert(bit32.rrotate(0x12345678, 32) == 0x12345678)`,
		},
		{
			"negative lrotate is rrotate",
			`assert(bit32.lrotate(1, -1) == 0x80000000)`,
		},
		{
			"negative rrotate is lrotate",
			`assert(bit32.rrotate(2, -1) == 4)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLuaSource(t, tt.source, tt.name)
		})
	}
}
