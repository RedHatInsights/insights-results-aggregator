#!/usr/bin/env bash

if [ $# -eq 0 ]
then
    echo
    echo "No arguments supplied. Expected prefix for benchmark function name."
    echo "For example 'Benchmark' to choose all of them"
    echo
    exit 1
fi

./benchmark.sh | grep -E "^$1" | sort -n -k 2,2
