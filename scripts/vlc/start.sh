#!/bin/bash
if [ -n "$1" ]
  then vlc --play-and-exit --fullscreen -I dummy "$1"
fi
