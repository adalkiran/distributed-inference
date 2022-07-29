#docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d --scale inference=4 --scale mediabridge=2
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d
