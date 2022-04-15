#!/bin/bash
GOOS=linux GOARCH=arm go build -ldflags "-s -w" -o x6100_cat_mux