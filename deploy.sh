#!/bin/bash

# Build for amd64
docker buildx build -t sukalov/karaokebot:latest --platform linux/amd64 .

# Push to Docker Hub
docker push sukalov/karaokebot:latest

# SSH and deploy
ssh root@147.45.163.30 << EOF
    #login
    docker login

    # pull the latest image
    docker pull sukalov/karaokebot:latest

    # remove existing container if it exists
    docker ps -a | grep karaokebot && docker stop karaokebot
    docker ps -a | grep karaokebot && docker rm karaokebot

    # run the new container
    docker run -d \
        --name karaokebot \
        -e BOT_TOKEN=${BOT_TOKEN} \
        -e ADMIN_BOT_TOKEN=${ADMIN_BOT_TOKEN} \
        sukalov/karaokebot:latest

EOF