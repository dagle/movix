local mp = require('mp')

local function onexit()
	local time = mp.get_property_native("playback-time")
	local path = mp.get_property_native("path")
	mp.commandv("run", "movix", "watched", path, time)
end

mp.add_hook("on_unload", 1, onexit)
