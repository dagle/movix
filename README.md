# movix
A database for your movies. Think a bit like netflix or plex but just the database for tracking what media you have and what you have seen.
The idea sprang from the fact that I don't need something like plex(with transcoding etc) but I want something where I can just say play next episode.
The media needs to formated in such a way that guessit can detect the content, it's not magic, the content needs to be named correctly.
Idealy you should pair movix with a program that will sort your media. For this I recommend [nmamer](https://github.com/jkwill87/mnamer)

TODO see the TODO file

## Install
First install guessit and golang, then type:

```
make && make install
```

## Config
To use movix, you first need to configure it. The config file is located at XDGHOME/movix/config
(defaults to ~/.config/movix/config). You need atleast this variable
```
	Mediapath: "/home/dagle/download/movix/"
```
Mediapath is where you want to keep all your media.

## Usage
After you have configured movix you can run:

```
movix create
```
Which will try to set things up in your mediapath

```
movix rescan
```
Will go over all your files and try to add them

Later on you can call:
```
movix add directory
```
OBS! Only add collection(like 1 tv-series) at the time with add and not a full library. Use rescan for that.

```
movix nexts
```

To get a list of series and movies with unwatched content, see movix-fzf and movix-rofi for examples to get a small menu.
