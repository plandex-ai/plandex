-- Module definition
local Example = {}

-- Constants
local MAX_RETRIES = 3
local DEFAULT_TIMEOUT = 5000

-- Private functions (local)
local function validateInput(input)
    if type(input) ~= "string" then
        error("Input must be a string")
    end
    return true
end

-- Metatable for creating classes
local function createClass(name)
    local cls = {}
    cls.__index = cls
    cls.__name = name
    
    -- Constructor
    function cls.new(...)
        local self = setmetatable({}, cls)
        if self.init then
            self:init(...)
        end
        return self
    end
    
    return cls
end

-- Class definition using metatables
local User = createClass("User")

function User:init(name, age)
    self.name = name
    self.age = age
    self.created_at = os.time()
end

function User:toString()
    return string.format("User(%s, %d)", self.name, self.age)
end

-- Table with custom metamethods
local DataStore = {
    data = {},
    __newindex = function(t, k, v)
        print("Setting value:", k, v)
        rawset(t.data, k, v)
    end,
    __index = function(t, k)
        return t.data[k]
    end
}
setmetatable(DataStore, DataStore)

-- Coroutine example
local function producer()
    return coroutine.create(function()
        for i = 1, 5 do
            coroutine.yield(i)
        end
    end)
end

-- Iterator function
local function range(from, to, step)
    step = step or 1
    local i = from - step
    return function()
        i = i + step
        if i <= to then
            return i
        end
    end
end

-- Module functions
function Example.process(input)
    assert(validateInput(input))
    
    local result = {
        original = input,
        processed = string.upper(input),
        timestamp = os.time()
    }
    
    return result
end

-- Function with multiple returns
function Example.divide(a, b)
    if b == 0 then
        return nil, "Division by zero"
    end
    return a / b
end

-- Closure example
function Example.counter(initial)
    local count = initial or 0
    return function()
        count = count + 1
        return count
    end
end

-- Table manipulation
function Example.merge(t1, t2)
    local result = {}
    for k, v in pairs(t1) do
        result[k] = v
    end
    for k, v in pairs(t2) do
        result[k] = v
    end
    return result
end

-- Pattern matching example
function Example.extractEmails(text)
    local emails = {}
    for email in string.gmatch(text, "[%w%.%-_]+@[%w%.%-_]+%.%w+") do
        table.insert(emails, email)
    end
    return emails
end

-- Event handling system
local EventEmitter = createClass("EventEmitter")

function EventEmitter:init()
    self.handlers = {}
end

function EventEmitter:on(event, handler)
    self.handlers[event] = self.handlers[event] or {}
    table.insert(self.handlers[event], handler)
end

function EventEmitter:emit(event, ...)
    if self.handlers[event] then
        for _, handler in ipairs(self.handlers[event]) do
            handler(...)
        end
    end
end

-- Add classes to module
Example.User = User
Example.EventEmitter = EventEmitter

-- Module return
return Example
