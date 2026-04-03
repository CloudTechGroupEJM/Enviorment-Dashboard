git pull origin main

docker rm -f cloud-assignment-2

docker build -t cloud-assignment-2 .

docker run -d -p 8080:8080 --name cloud-assignment-2 cloud-assignment-2