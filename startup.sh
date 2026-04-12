git pull origin main

docker rm -f cloud-assignment-2
docker build -t cloud-assignment-2 .
docker compose -f cloud-assignment-2.yaml -p cloud-assignment-2 up -d