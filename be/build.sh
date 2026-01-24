#! /bin/bash

mkdir -p output/conf
cp conf/deploy.local.yml output/conf

go build -o output/main