local mp = require('mp')

local function onexit()
	local time = mp.get_property_native("playback-time")
	local path = mp.get_property_native("path")
	mp.commandv("run", "movix", "watched", path, time)
end

mp.add_hook("on_unload", 1, onexit)
-- we want to add a function to exit without saving
-- we want to add a function to mark this as watched (and quit?)
