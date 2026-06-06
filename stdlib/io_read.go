package stdlib

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/iceisfun/golua/v2/vm"
)

// This file implements the f:read(...) family: format dispatch and the
// numeric-format parsing (parseReadNumber / parseHexFloat) used by the "n"
// read format. Split out of io.go.

// fileRead implements f:read(...) method.
func fileRead(v *vm.VM) int {
	fh := getFileHandle(v, v.Get(1), "read")
	fh.checkOpen(v.Context(), "read")

	return doFileRead(v, fh.file, 2)
}

// doFileRead performs the actual read operation using arguments from the stack.
// firstArg is the index of the first format argument (2 for method calls, 1 for io.read).
func doFileRead(v *vm.VM, f vm.LuaFile, firstArg int) int {
	n := v.ArgCount() - (firstArg - 1) // number of format args
	var formats []vm.Value
	if n > 0 {
		formats = make([]vm.Value, n)
		for i := 0; i < n; i++ {
			formats[i] = v.Get(firstArg + i)
		}
	}
	return doFileReadFormats(v, f, formats, firstArg)
}

// doFileReadFormats performs the actual read operation using a slice of formats.
// firstArg controls error reporting: 1 for io.read (module func), 2 for f:read (method).
func doFileReadFormats(v *vm.VM, f vm.LuaFile, formats []vm.Value, firstArg int) int {
	ctx := v.Context()
	if len(formats) == 0 {
		// Default: read a line
		line, err := f.Read(ctx, "l")
		if err != nil {
			if err == io.EOF {
				v.Set(0, vm.Nil)
				return 1
			}
			errno, errDesc := extractLuaFileError(err)
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(errDesc))
			v.Set(2, vm.NewInt(int64(errno)))
			return 3
		}
		v.Set(0, vm.NewString(line))
		return 1
	}

	// Process each format argument.
	// Track OS-level I/O errors: Lua 5.4 checks ferror(f) after the loop and
	// returns nil, errmsg, errno if the underlying file had an error.
	// Normal failures (EOF, no-number-found) produce nil and stop further reads
	// (Lua 5.4 uses a success flag that gates the loop).
	n := len(formats)
	v.EnsureStack(v.Base() + n)
	results := 0
	var fileErr error
	success := true
	for _, arg := range formats {
		if !success {
			v.Set(results, vm.Nil)
			results++
			continue
		}
		if arg.IsNumber() {
			// Read N bytes
			count, ok := arg.ToInt()
			if !ok {
				fileArgError(v, firstArg+results, "read", "number has no integer representation")
			}
			if count < 0 {
				panic("resulting string too large")
			}
			// Bound count before delegating to provider's allocator. Without
			// this, very large counts (e.g. 1<<60) reach make([]byte, count)
			// and trigger a Go runtime makeslice panic ("len out of range")
			// that leaks through pcall. Reference Lua raises the structured
			// "not enough memory" error.
			const maxReadBytes = 1 << 30 // 1 GiB
			if count > maxReadBytes {
				panic("not enough memory")
			}
			if count == 0 {
				// Read 0 bytes: test if at EOF
				data, err := f.ReadBytes(ctx, 0)
				if err != nil {
					if isFileError(err) {
						fileErr = err
					}
					v.Set(results, vm.Nil)
					success = false
				} else {
					v.Set(results, vm.NewString(data))
				}
			} else {
				data, err := f.ReadBytes(ctx, int(count))
				if err != nil {
					if isFileError(err) {
						fileErr = err
					}
					v.Set(results, vm.Nil)
					success = false
				} else {
					v.Set(results, vm.NewString(data))
				}
			}
		} else if arg.IsString() {
			format := arg.AsString()
			// Validate format before calling provider
			cleanFmt := strings.TrimPrefix(format, "*")
			if len(cleanFmt) == 0 || (cleanFmt[0] != 'a' && cleanFmt[0] != 'l' && cleanFmt[0] != 'L' && cleanFmt[0] != 'n') {
				// Use results+1 as the user-visible argument index (1-based, for the format arg)
				fileArgError(v, firstArg+results, "read", "invalid format")
			}
			data, err := f.Read(ctx, format)
			if err != nil {
				if isFileError(err) {
					fileErr = err
				}
				v.Set(results, vm.Nil)
				success = false
			} else {
				// Check if format is "n" or "*n" for number parsing
				cleanFmt := strings.TrimPrefix(format, "*")
				if len(cleanFmt) > 0 && cleanFmt[0] == 'n' {
					// Parse as number matching Lua 5.4 semantics:
					// - Leading zeros are decimal (not octal)
					// - Hex floats (0x1.8, 0xABp0) produce floats
					// - Overflow to ±Inf is valid
					v.Set(results, parseReadNumber(data))
				} else {
					v.Set(results, vm.NewString(data))
				}
			}
		} else {
			// Invalid format type
			fileArgError(v, firstArg+results, "read", fmt.Sprintf("string expected, got %s", v.ObjTypeName(arg)))
		}
		results++
	}
	// Lua 5.4: if the file had an OS-level I/O error (ferror), return nil, msg, errno.
	if fileErr != nil {
		errno, errDesc := extractLuaFileError(fileErr)
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(errDesc))
		v.Set(2, vm.NewInt(int64(errno)))
		return 3
	}
	return results
}

// parseReadNumber converts a string read by read("n") to a Lua number value.
// Matches Lua 5.4 semantics: leading zeros are decimal (not octal), hex floats
// (0x1.8, 0xABp0) produce floats, and overflow to ±Inf is valid.
func parseReadNumber(data string) vm.Value {
	if len(data) == 0 {
		return vm.Nil
	}

	// Determine sign and hex prefix, accounting for optional leading sign.
	digitStart := 0
	if len(data) > 0 && (data[0] == '+' || data[0] == '-') {
		digitStart = 1
	}
	isHex := len(data) > digitStart+2 && data[digitStart] == '0' && (data[digitStart+1] == 'x' || data[digitStart+1] == 'X')
	hexBody := ""
	if isHex && len(data) > digitStart+2 {
		hexBody = data[digitStart+2:]
	}
	isHexFloat := isHex && strings.ContainsAny(hexBody, ".pP")
	// For hex numbers, only '.', 'p', 'P' indicate float (not 'e'/'E' which are hex digits).
	hasFloatIndicator := !isHex && strings.ContainsAny(data, ".eE")

	// Hex floats always produce float type
	if isHexFloat {
		fv, err := strconv.ParseFloat(data, 64)
		if err != nil {
			// Accept overflow/underflow results (ErrRange) — produces ±Inf or 0.
			if numErr, ok := err.(*strconv.NumError); ok && numErr.Err == strconv.ErrRange {
				return vm.NewFloat(fv)
			}
			// Go doesn't support hex floats without p exponent (e.g. "0x1.8").
			// Parse manually: integer part + fractional part.
			fv, ok := parseHexFloat(data)
			if !ok {
				return vm.Nil
			}
			return vm.NewFloat(fv)
		}
		return vm.NewFloat(fv)
	}

	// Try integer first (base 10 for decimal, base 0 for hex with prefix)
	if !hasFloatIndicator {
		if isHex {
			// Use base 0 which handles [+-]0x prefix natively
			if iv, err := strconv.ParseInt(data, 0, 64); err == nil {
				return vm.NewInt(iv)
			}
			// Fallback: try unsigned parse for values > max int64 (e.g. 0x8000000000000001)
			// then wrap to int64, matching Lua 5.4's lua_stringtonumber behavior.
			sign := int64(1)
			hexStr := hexBody
			if digitStart > 0 && data[0] == '-' {
				sign = -1
			}
			if u, err := strconv.ParseUint(hexStr, 16, 64); err == nil {
				return vm.NewInt(sign * int64(u))
			}
			// Overflow: parse digit-by-digit with modular wrapping (Lua 5.5
			// lexer/tonumber semantics for hex int literals exceeding 2^64).
			if hexStr != "" {
				var result uint64
				valid := true
				for _, c := range hexStr {
					var d uint64
					switch {
					case c >= '0' && c <= '9':
						d = uint64(c - '0')
					case c >= 'a' && c <= 'f':
						d = uint64(c-'a') + 10
					case c >= 'A' && c <= 'F':
						d = uint64(c-'A') + 10
					default:
						valid = false
					}
					if !valid {
						break
					}
					result = result*16 + d
				}
				if valid {
					return vm.NewInt(sign * int64(result))
				}
			}
		} else {
			if iv, err := strconv.ParseInt(data, 10, 64); err == nil {
				return vm.NewInt(iv)
			}
		}
	}

	// Try float
	fv, err := strconv.ParseFloat(data, 64)
	if err != nil {
		// Accept overflow results (ErrRange) — produces ±Inf
		if numErr, ok := err.(*strconv.NumError); ok && numErr.Err == strconv.ErrRange {
			if math.IsInf(fv, 0) {
				return vm.NewFloat(fv)
			}
		}
		return vm.Nil
	}

	return vm.NewFloat(fv)
}

// parseHexFloat parses hex floats without p exponent (e.g. "0x1.8" = 1.5).
// Go's strconv.ParseFloat requires p exponent for hex floats.
func parseHexFloat(data string) (float64, bool) {
	s := data
	neg := false
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		s = s[1:]
	}
	if len(s) < 3 || s[0] != '0' || (s[1] != 'x' && s[1] != 'X') {
		return 0, false
	}
	s = s[2:]

	// Split at dot
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = "" // will be set below
		rest := parts[1]
		// Check for p/P exponent
		pIdx := strings.IndexAny(rest, "pP")
		if pIdx >= 0 {
			// Has exponent — let Go handle it by appending p0
			withP := data
			if !strings.ContainsAny(data, "pP") {
				withP = data + "p0"
			}
			fv, err := strconv.ParseFloat(withP, 64)
			return fv, err == nil
		}
		fracPart = rest
	}

	// Parse integer part
	var result float64
	if intPart != "" {
		iv, err := strconv.ParseUint(intPart, 16, 64)
		if err != nil {
			return 0, false
		}
		result = float64(iv)
	}

	// Parse fractional part
	if fracPart != "" {
		frac := 0.0
		scale := 1.0 / 16.0
		for _, c := range fracPart {
			var d int
			switch {
			case c >= '0' && c <= '9':
				d = int(c - '0')
			case c >= 'a' && c <= 'f':
				d = int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				d = int(c-'A') + 10
			default:
				return 0, false
			}
			frac += float64(d) * scale
			scale /= 16.0
		}
		result += frac
	}

	if neg {
		result = -result
	}
	return result, true
}
