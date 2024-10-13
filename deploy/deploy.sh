#!/bin/sh

# check if the first argument is set
if [ -z "$1" ]; then
    echo "Please provide the IP/Hostname of the server to depoy to as the first argument"
    exit 1
fi



SERVER_HOSTNAME=$1

echo "Building image"

docker build -t smartmeter .

docker save -o smartmeter.tar smartmeter

echo "Sending image to server"
scp smartmeter.tar $SERVER_HOSTNAME:~/smartmeter.tar
scp deploy/run-on-server.sh $SERVER_HOSTNAME:~/run-on-server.sh
scp backend/config.yaml $SERVER_HOSTNAME:~/config.yaml

ssh $SERVER_HOSTNAME chmod +x run-on-server.sh
ssh $SERVER_HOSTNAME ./run-on-server.sh

du -sh  smartmeter.tar
rm smartmeter.tar