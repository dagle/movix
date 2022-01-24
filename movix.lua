local mp = require('mp')

local save = true
local function onexit()
	local time = mp.get_property_native("playback-time")
	local path = mp.get_property_native("path")
	if save then
		print("Saving watched content to db: " .. path)
		mp.commandv("run", "movix", "watched", path, time)
	end
end

local function nosave()
	save = false
	mp.commandv("quit")
end

local function skip()
	print("Skipping episode/movie")
	local path = mp.get_property_native("path")
	mp.commandv("run", "movix", "watched", path, "full")
	nosave()
end

mp.add_hook("on_unload", 1, onexit)
mp.add_forced_key_binding("Q", "skip", skip)
mp.add_key_binding("Ctrl+q", "nosave", nosave)
-- we want to add a function to exit without saving
-- we want to add a function to mark this as watched (and quit?)
