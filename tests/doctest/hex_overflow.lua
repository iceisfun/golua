-- Hex overflow wraps to low 64 bits (integer)
print(type(0x10000000000000000))
--> =number
print(math.type(0x10000000000000000))
--> =integer
print(0x10000000000000000)
--> =0
print(0x10000000000000001)
--> =1
print(0x1FFFFFFFFFFFFFFFF)
--> =-1

-- Decimal overflow stays float (unchanged)
print(math.type(99999999999999999999))
--> =float

-- Bitwise on wrapped values
print(0x13121110090807060504030201 & 0xFFFF)
--> =513

-- string.format %X on wrapped values
print(string.format("0x%X", 0x10000000000000000))
--> =0x0
print(string.format("0x%X", 0x13121110090807060504030201))
--> =0x807060504030201
