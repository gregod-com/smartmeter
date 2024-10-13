#!/bin/sh

sudo docker ps

sudo docker load -i smartmeter.tar
sudo docker stop smartmeter || true
sudo docker rm smartmeter || true
sleep 2
sudo docker run -d \
    --name smartmeter \
    -v /home/gregor/gorm.db:/app/gorm.db \
    -v /home/gregor/config.yaml:/app/config.yaml \
    -p 80:8080 \
    smartmeter
