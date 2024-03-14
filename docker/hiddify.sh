#!/bin/sh
if [ -f "/opt/hiddify.json" ]; then
    /hiddify/HiddifyCli run --config "$CONFIG" -h /hiddify/data/hiddify.json
else
    /hiddify/HiddifyCli run --config "$CONFIG"
fi
