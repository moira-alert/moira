#!/bin/sh

# todo: fix path to bin
cd ./database
/home/andrey/Dev/gowrap/bin/go_build_gowrap_go gen -p .. -i Database -t ../metrics/gen_templates/moira-metrics -o database_with_metrics.go
