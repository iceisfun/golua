-- test_math_regression.lua
-- Numerical / runtime regression test promoted from the original a.lua probe.
-- Exercises:
--   * vec4 metatable (__add, __sub, __mul scalar+vector overloads, __tostring)
--   * quaternion __mul + axis/angle construction (sin/cos/sqrt usage)
--   * mat4 multiplication and transform application
--   * chained transforms (translation * rotation * scale)
--   * iterative renormalization drift over 1000 rotations
--   * integer/float distinction preservation through arithmetic
--
-- Reference values were captured at full %.17g precision from lua5.4.8 and
-- lua5.5.0 (both produce identical output) and are used as the expected
-- values below. Tolerances are loose enough to absorb a different libm's
-- last-bit wiggle but tight enough to catch real metamethod / arithmetic
-- regressions (e.g. operand swap in __mul, missed __add dispatch, NaN
-- propagation, sign inversion).

local EPSILON = 1e-9

local function approx(got, want, tol)
    local d = got - want
    if d < 0 then d = -d end
    return d <= tol
end

local function close(got, want, tol, label)
    if not approx(got, want, tol) then
        error(string.format("%s: got %.17g want %.17g (|diff|=%.3g > tol %.3g)",
            label, got, want, math.abs(got - want), tol), 2)
    end
end

local function close_vec(got, wx, wy, wz, ww, tol, label)
    close(got.x, wx, tol, label .. ".x")
    close(got.y, wy, tol, label .. ".y")
    close(got.z, wz, tol, label .. ".z")
    close(got.w, ww, tol, label .. ".w")
end

----------------------------------------------------------------
-- vec4
----------------------------------------------------------------

local vec4 = {}
vec4.__index = vec4

function vec4.new(x, y, z, w)
    return setmetatable({
        x = x or 0,
        y = y or 0,
        z = z or 0,
        w = w or 0,
    }, vec4)
end

function vec4.__add(a, b)
    return vec4.new(a.x + b.x, a.y + b.y, a.z + b.z, a.w + b.w)
end

function vec4.__sub(a, b)
    return vec4.new(a.x - b.x, a.y - b.y, a.z - b.z, a.w - b.w)
end

function vec4.__mul(a, b)
    if type(a) == "number" then
        return vec4.new(a * b.x, a * b.y, a * b.z, a * b.w)
    elseif type(b) == "number" then
        return vec4.new(a.x * b, a.y * b, a.z * b, a.w * b)
    else
        return a.x * b.x + a.y * b.y + a.z * b.z + a.w * b.w
    end
end

function vec4:length()
    return math.sqrt(self * self)
end

function vec4:normalize()
    local len = self:length()
    if len < EPSILON then
        return vec4.new(0, 0, 0, 0)
    end
    return self * (1.0 / len)
end

function vec4:cross(b)
    return vec4.new(
        self.y * b.z - self.z * b.y,
        self.z * b.x - self.x * b.z,
        self.x * b.y - self.y * b.x,
        0)
end

----------------------------------------------------------------
-- quaternion
----------------------------------------------------------------

local quat = {}
quat.__index = quat

function quat.new(x, y, z, w)
    return setmetatable({
        x = x or 0,
        y = y or 0,
        z = z or 0,
        w = w or 1,
    }, quat)
end

function quat.from_axis_angle(axis, angle)
    local half = angle * 0.5
    local s = math.sin(half)
    axis = axis:normalize()
    return quat.new(axis.x * s, axis.y * s, axis.z * s, math.cos(half))
end

function quat.__mul(a, b)
    return quat.new(
        a.w * b.x + a.x * b.w + a.y * b.z - a.z * b.y,
        a.w * b.y - a.x * b.z + a.y * b.w + a.z * b.x,
        a.w * b.z + a.x * b.y - a.y * b.x + a.z * b.w,
        a.w * b.w - a.x * b.x - a.y * b.y - a.z * b.z)
end

function quat:normalize()
    local len = math.sqrt(
        self.x * self.x + self.y * self.y +
        self.z * self.z + self.w * self.w)
    if len < EPSILON then
        return quat.new()
    end
    return quat.new(self.x / len, self.y / len, self.z / len, self.w / len)
end

function quat:to_mat4()
    local x, y, z, w = self.x, self.y, self.z, self.w
    local xx, yy, zz = x * x, y * y, z * z
    local xy, xz, yz = x * y, x * z, y * z
    local wx, wy, wz = w * x, w * y, w * z
    return {
        1 - 2 * (yy + zz), 2 * (xy - wz),     2 * (xz + wy),     0,
        2 * (xy + wz),     1 - 2 * (xx + zz), 2 * (yz - wx),     0,
        2 * (xz - wy),     2 * (yz + wx),     1 - 2 * (xx + yy), 0,
        0, 0, 0, 1,
    }
end

----------------------------------------------------------------
-- mat4
----------------------------------------------------------------

local mat4 = {}
mat4.__index = mat4

function mat4.new(values)
    return setmetatable({ m = values }, mat4)
end

function mat4.translation(x, y, z)
    return mat4.new({
        1, 0, 0, x,
        0, 1, 0, y,
        0, 0, 1, z,
        0, 0, 0, 1,
    })
end

function mat4.scale(x, y, z)
    return mat4.new({
        x, 0, 0, 0,
        0, y, 0, 0,
        0, 0, z, 0,
        0, 0, 0, 1,
    })
end

function mat4.from_quat(q)
    return mat4.new(q:to_mat4())
end

function mat4.__mul(a, b)
    local out = {}
    for row = 0, 3 do
        for col = 0, 3 do
            local sum = 0
            for k = 0, 3 do
                sum = sum + a.m[row * 4 + k + 1] * b.m[k * 4 + col + 1]
            end
            out[row * 4 + col + 1] = sum
        end
    end
    return mat4.new(out)
end

function mat4:transform(v)
    local m = self.m
    return vec4.new(
        m[1]  * v.x + m[2]  * v.y + m[3]  * v.z + m[4]  * v.w,
        m[5]  * v.x + m[6]  * v.y + m[7]  * v.z + m[8]  * v.w,
        m[9]  * v.x + m[10] * v.y + m[11] * v.z + m[12] * v.w,
        m[13] * v.x + m[14] * v.y + m[15] * v.z + m[16] * v.w)
end

----------------------------------------------------------------
-- workload (verbatim from the original a.lua)
----------------------------------------------------------------

local axis = vec4.new(0.3, 0.7, 0.2, 0):normalize()

local q =
    quat.from_axis_angle(axis, math.rad(37.0)) *
    quat.from_axis_angle(vec4.new(1, 0, 0, 0), math.rad(12.5))
q = q:normalize()

local rotation    = mat4.from_quat(q)
local translation = mat4.translation(10.5, -4.25, 8.75)
local scale       = mat4.scale(1.25, 0.5, -2.0)
local transform   = translation * rotation * scale

local points = {
    vec4.new(1, 2, 3, 1),
    vec4.new(-4, 0.5, 8, 1),
    vec4.new(0.25, -9, 1.5, 1),
}

local tp    = {}
local accum = vec4.new()
for i, p in ipairs(points) do
    tp[i] = transform:transform(p)
    accum = accum + tp[i]
end

local a     = vec4.new(1, 2, 3, 0)
local b     = vec4.new(-2, 0.5, 7, 0)
local dot   = a * b
local cross = a:cross(b)
local na    = a:normalize()
local nb    = b:normalize()

local drift = vec4.new(1, 0, 0, 1)
for _ = 1, 1000 do
    drift = rotation:transform(drift)
    drift = drift:normalize()
end

local summary =
    accum.x * 0.123 +
    accum.y * 0.456 +
    accum.z * 0.789 +
    drift.x * 1.111 +
    drift.y * 2.222 +
    drift.z * 3.333

----------------------------------------------------------------
-- assertions (reference values from lua5.4.8 / lua5.5.0)
----------------------------------------------------------------

local FLOAT_TOL = 1e-12  -- single-shot float math
local DRIFT_TOL = 1e-9   -- 1000 iterated rotations accumulate ULPs

-- transform matrix
local M = transform.m
close(M[1],   1.0348322990424701,    FLOAT_TOL, "M[1]")
close(M[2],   0.018682810272097471,  FLOAT_TOL, "M[2]")
close(M[3],  -1.1193604649634039,    FLOAT_TOL, "M[3]")
assert(M[4] == 10.5,  "M[4] preserves translation x exactly")
close(M[5],   0.27633158772182481,   FLOAT_TOL, "M[5]")
close(M[6],   0.44764440088569368,   FLOAT_TOL, "M[6]")
close(M[7],   0.77353243692587359,   FLOAT_TOL, "M[7]")
assert(M[8] == -4.25, "M[8] preserves translation y exactly")
close(M[9],  -0.64440900559009207,   FLOAT_TOL, "M[9]")
close(M[10],  0.22195820092086072,   FLOAT_TOL, "M[10]")
close(M[11], -1.465837548468599,     FLOAT_TOL, "M[11]")
assert(M[12] == 8.75, "M[12] preserves translation z exactly")
assert(M[13] == 0 and M[14] == 0 and M[15] == 0 and M[16] == 1,
    "M bottom row must remain exactly 0,0,0,1")

-- transformed points (homogeneous w stays exactly 1)
close_vec(tp[1],
     8.214116524696454,  -0.757782299729167,
     4.1519947508458319,  1, FLOAT_TOL, "tp[1]")
close_vec(tp[2],
    -2.5848715107410634,  1.0567553449625366,
    -0.28808526492799302, 1, FLOAT_TOL, "tp[2]")
close_vec(tp[3],
     8.9115220848666343, -7.0494180556519765,
     4.3925176176118317,  1, FLOAT_TOL, "tp[3]")
assert(tp[1].w == 1 and tp[2].w == 1 and tp[3].w == 1,
    "homogeneous w must remain exactly 1 (bottom row is 0,0,0,1)")

-- vector ops: these inputs make every component exactly representable
assert(dot == 20.0,    "dot product should be exact 20.0")
assert(math.type(dot) == "float", "dot should be float (mixed int/float operands)")
assert(cross.x == 12.5, "cross.x")
assert(cross.y == -13,  "cross.y")
assert(cross.z == 4.5,  "cross.z")
assert(cross.w == 0,    "cross.w")
assert(math.type(cross.y) == "integer",
    "cross.y stays integer (3*-2 - 1*7, all int operands)")
assert(math.type(cross.w) == "integer",
    "cross.w stays integer (literal 0 from constructor)")

close_vec(na,
     0.26726124191242440, 0.53452248382484879,
     0.80178372573727319, 0, FLOAT_TOL, "norm(a)")
close_vec(nb,
    -0.27407548393101266, 0.068518870982753166,
     0.95926419375854433, 0, FLOAT_TOL, "norm(b)")

-- normalized vectors must have unit length
local function vec_len(v)
    return math.sqrt(v.x*v.x + v.y*v.y + v.z*v.z + v.w*v.w)
end
close(vec_len(na), 1.0, FLOAT_TOL, "|norm(a)|")
close(vec_len(nb), 1.0, FLOAT_TOL, "|norm(b)|")

-- iterative drift (1000 rotations + renormalizations)
close_vec(drift,
     0.35303418323258834,  0.358440739401582,
    -0.49687735086897045,  0.70710678118653392,
    DRIFT_TOL, "drift")
close(vec_len(drift), 1.0, 1e-12, "|drift|")

-- accumulated points
close(accum.x,  14.540767098822025, FLOAT_TOL, "accum.x")
close(accum.y,  -6.7504450104186073, FLOAT_TOL, "accum.y")
close(accum.z,   8.2564271035296706, FLOAT_TOL, "accum.z")
assert(accum.w == 3, "accum.w should be exactly 3 (sum of three integer 1s)")

-- hash-like summary
close(summary, 4.7572165031645763, 1e-12, "summary")
