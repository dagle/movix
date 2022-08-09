#!/usr/bin/bash

# This doesn't let you seed because it might create a directory
# Maybe rewrite this in python or something to make that work

HOST=localhost
PORT=9091
# USER=YOUR_USER_NAME
# PASS=YOUR_PASSWORD

# REMOTE="transmission-remote $HOST:$PORT --auth=$USER:$PASS"
REMOTE="transmission-remote $HOST:$PORT"

# remove -S
FILES=$($REMOTE -t $TR_TORRENT_ID -f | tail -n +4 | awk -F ' ' '{print $7}')
ROOTFILES=$(for file in $(echo $FILES); do echo $file; done | perl -pe 's|(.*?/).*|\1|' | uniq)
NEWDIR=$(echo $ROOTFILES | wc -l)

if [ $NEWDIR -gt 1 ]; then
	DIR=${TR_TORRENT_DIR}/${TR_TORRENT_NAME}
	mkdir $DIR
	$REMOTE -t $TR_TORRENT_ID --move $DIR
else
	# maybe like this
	DIR=$TR_TORRENT_DIR/${ROOTFILES}
fi

# echo movix add "$DIR"
/home/dagle/code/movix/movix add "$DIR" 2>&1 >> /home/dagle/transmission.log
echo $? >> /home/dagle/transmission.log
# $REMOTE -t $TR_TORRENT_ID -rad
