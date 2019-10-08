#!/usr/bin/env bash

pid=`ps aux | grep -v grep | grep ahri-server | awk '{print $2}'`

if [[ $pid != '' ]]; then
	kill $pid
fi

exit 0
