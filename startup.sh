git pull origin main

docker rm -f assignment-2-app-1
docker compose build
docker compose up -d