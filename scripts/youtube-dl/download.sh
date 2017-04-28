#!/bin/bash
if [ -n "$1" ]
  then youtube-dl -o "$1" "$2"
fi
