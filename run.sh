#!/bin/sh

go build -o bin/infra-sanity

./bin/infra-sanity hypershift-ppc64le.com ibm-hypershift-dev one
