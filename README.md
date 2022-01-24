# movix
A database for your movies. Think a bit like netflix or plex but just the database for tracking what media you have and what you have seen.
The idea sprang from the fact that I don't need something like plex(with transcoding etc) but I want something where I can just say play next episode.
The media needs to formated in such a way that guessit can detect the content, it's not magic, the content needs to be named correctly.

TODO see the TODO file

## Install
```
make install
```

## Config
To use movix, you first need to configure it. The config file is located at XDGHOME/movix/config
(defaults to ~/.config/movix/config). You need atleast these 2 variables in the config 
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
movix add file

```
To move a file to mediadir and add it to the db. 

```
movix del file
```





You can then run
```
movix nexts
```

To get a list of series and movies with unwatched content 


