git pull origin main

docker rm -f cloud-assignment-2
docker build -t cloud-assignment-2 .
docker run -d -p 8080:8080 \
 --name cloud-assignment-2 \
  -v "$PWD/credentials:/app/credentials:ro" \
  cloud-assignment-2