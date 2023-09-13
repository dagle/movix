#!/bin/sh

media-sort -r -h --tv-template='{{ .Name }}/{{ .Name }} S{{ printf "%02d" .Season }}E{{ printf "%02d" .Episode }}{{ if ne .ExtraEpisode -1 }}-{{ printf "%02d" .ExtraEpisode }}{{end}}.{{ .Ext }}' -t /mnt/tv -m /mnt/movie
