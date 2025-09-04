#!/bin/bash
echo $1 $2
file_name=$1
client_number=$2
cat >$file_name <<EOL
name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - EXPECTED_AGENCIES=1
    networks:
      - testing_net
    volumes:
      - ./server/config.ini:/config.ini
EOL
for i in $(seq 1 $client_number);
do
cat >>$file_name <<EOL

  client${i}:
    container_name: client${i}
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=${i}
    networks:
      - testing_net
    depends_on:
      - server
    volumes:
      - ./client/config.yaml:/config.yaml
      - ./.data/agency-${i}.csv:/dataset.csv
EOL
done

cat >>${file_name} << EOL

networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
EOL