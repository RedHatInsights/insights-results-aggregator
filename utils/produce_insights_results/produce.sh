#!/bin/bash

while true
do
    for file in *.json
    do
        echo $file
        kafkacat -b localhost:9092 -P -t ccx.ocp.results $file
        sleep 1
    done
done
