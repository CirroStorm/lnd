#!/bin/bash

protoc -I/usr/local/include -I. --go_out=plugins=grpc,source_relative:. hwwallet.proto