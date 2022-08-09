# movix
A database for your movies. Think a bit like netflix or plex but just the database for tracking what media you have and what you have seen.
The idea sprang from the fact that I don't need something like plex(with transcoding etc) but I want something where I can just say play next episode or list unwatched movies
The media needs to formated in such a way that guessit can detect the content, it's not magic, the content needs to be named correctly. 

## Install
First install guessit and golang, then type:

```
make && make install
```

## Config
On startup movix will read config file at $XDG_CONFIG_HOME/movix/config.yml if
such an executable exists. If $XDG_CONFIG_HOME is not set, ~/.config/movix/config.yml
will be used instead.

## Usage
After you have configured movix you can run:

```
movix create
```
To add a directory do:
```
movix add directory
```
Only add collection(like 1 tv-series or movie) at the time with add and not a full library.
The later might work but hasn't been tested

```
movix suggest
```

To get a list of series and movies with unwatched content, see movix-fzf and movix-rofi for examples to get a small menu.

Movix comes with a lua script for mpv that reports back to movix how much you have watched, it's invoked atomatically when you launch mpv through movix.

Movix comes with a transmission script to automatically add new content from torrents, to be ran at torrent complition
