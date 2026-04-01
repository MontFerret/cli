#!/bin/bash

if [ "$1" = 'ferret' ]; then
  echo $(cat go.mod | grep 'github.com/MontFerret/ferret/' | awk '{print $2}' | sed 's/^v//')
else
  echo $(git describe --tags --always --dirty)
fi
