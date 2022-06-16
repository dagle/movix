#!/usr/bin/env bash

# This doesn't let you seed, a "later" problem.
# Maybe rewrite this in python or something to make that work

HOST=localhost
PORT=9091
# USER=YOUR_USER_NAME
# PASS=YOUR_PASSWORD

# REMOTE="transmission-remote $HOST:$PORT --auth=$USER:$PASS"
REMOTE="transmission-remote $HOST:$PORT"

FILES=$($REMOTE -t $TR_TORRENT_ID -S -f | tail -n +4 | awk -F ' ' '{print $7}')
ROOTFILES=$(echo $FILES | perl -pe 's|(.*?/).*|\1|')
ROOT=$(echo $FILES | uniq)
NEWDIR=$(echo $ROOT | wc -l)

if [ $NEWDIR -gt 1 ] then
	DIR=$TR_TORRENT_DIR/$TR_TORRENT_NAME
	mkdir $DIR
	$REMOTE -t $TR_TORRENT_ID --move $DIR
else
	DIR=$TR_TORRENT_DIR/$ROOT
fi

movix add $DIR
$REMOTE -t $TR_TORRENT_ID -rad
