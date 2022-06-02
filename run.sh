#!/bin/sh

go build -o bin/infra-sanity

./bin/infra-sanity scnl-ibm.com hypershift-resource all
