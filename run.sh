#!/bin/bash

# git clone https://github.com/amsehili/auditok.git
# cd audiotok
# python setup.py install

# pip install pydub
# apt-get install -y sox

# lsmod
export GOOGLE_APPLICATION_CREDENTIALS="/home/calum/Downloads/WillowBot-e183b1183bb5.json"
export AUDIODEV="hw:CARD=P780,DEV=0"

mkdir -p output/

willow &

rec -q -t raw -r 16000 -c 1 -b 16 -e signed - | auditok -i - -e 70 -o output/audiotok-{start}-{end}.flac
