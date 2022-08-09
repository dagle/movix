movix(1) "github.com/dagle/movix" "General Commands Manual"

# NAME

movix - the unix movie database

# SYNOPSIS

*movix* init
*movix* add directory
*movix* watched path full|time
*movix* next name
*movix* suggest
*movix* skip name season episode

# DESCRIPTION

*movix* is a database to keep track of movies and tv-series, or other type of
media that you typically tend to watch once.

It keeps track of media you have on harddrive, group episodes into categories, 
where you stopped watching particular media.

# add
scan a *directory* for media to add to the database

# watched 
mark the entry with *path* in the db as watched. Use either time to mark the
amount of time watched (in seconds) or full to mark that the whole file has been watched.

# next 
get the next entry with the *name*. Useful when working with something containing
series, this way you can say *movix* next series-name 

# suggest
Gets the names of series and movies that has unwatched content.

# skip
Skip until *season* *episode* for series with *name*

# CONFIGURATION

On startup movix will read config file at $XDG_CONFIG_HOME/movix/config.yml if
such an executable exists. If $XDG_CONFIG_HOME is not set, ~/.config/movix/config.yml
will be used instead.


# AUTHORS
Maintained by per odlund <per.odlund@gmail.com>
