-- hex float strings should convert to integer when exact
print(math.tointeger("0xff.0"))
--> 255
print(math.tointeger("0x10.0"))
--> 16
print(math.tointeger("0xff.1"))
--> nil
print(string.format("%d", "0xff.0"))
--> 255
