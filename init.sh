#!/usr/bin/env bash
set -eux
#set -e
# Updating queue.conf file
REPLACEMENT="command=${PWD}/queue.sh" &&\
sed --expression "s@command=queue.sh@$REPLACEMENT@" ./queue.conf

sudo cp -rf queue.conf /etc/supervisor/conf.d/queue.conf
#sudo cp queue.conf
exit 0