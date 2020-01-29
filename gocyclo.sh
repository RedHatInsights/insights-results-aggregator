#!/bin/bash

go get github.com/fzipp/gocyclo
gocyclo -over 9 -avg .
