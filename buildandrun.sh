#!/bin/bash
harbor='120.24.46.111:2189'
appName='mqbus'
version=v1.0.0
repo=test
imageName=${harbor}/${repo}/${appName}:${version}
port=10001

echo ---------------docker stop...------------------
docker stop ${appName}

echo ---------------docker rm...------------------
docker rm ${appName}

echo ---------------docker rmi ...------------------
docker rmi ${harbor}/${repo}/${appName}:${version}

echo ---------------docker build ...------------------
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./${appName} ./${appName}.go
docker build -t ${imageName} .
docker login -u admin -p Harbor12345 http://120.24.46.111:2189
docker push ${imageName}

echo ---------------docker run ...------------------
docker run --name ${appName} -d --restart=always -v e:/logs:/home/logs ${harbor}/${repo}/${appName}:${version}