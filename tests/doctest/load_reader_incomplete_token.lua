local i = 0
local f, err = load(function()
    i = i + 1
    if i <= 99 then
        return " "
    end
    if i == 100 then
        return "return 0x"
    end
    if i == 101 then
        return "1"
    end
end)

print(type(f))
--> =function

print(err == nil)
--> =true

print(f())
--> =1
