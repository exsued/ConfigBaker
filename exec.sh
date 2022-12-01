#!/bin/bash

docker rm go-web-app && docker run --name=go-web-app -p 8080:8080 go-app
