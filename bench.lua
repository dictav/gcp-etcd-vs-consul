local requests = { }
local total = 0

init = function(args)
   wrk.headers["User-Agent"] = "wrk benchmark tool"

   local f = io.open("blacklist.txt", "r")
   for path in f:lines() do
      table.insert(requests, wrk.format(nil, '/' .. path))
   end
   f:close()

   total = table.maxn(requests)
end

request = function()
   local n = math.random(total)
   return requests[n]
end
