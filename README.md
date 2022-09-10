# redis-poc

`docker-compose` will start:
- a redis server with [RedisInsight](https://redis.com/redis-enterprise/redis-insight/)
- a micro-service build with golang for basic CRUD purposes (Rest API)

## :rocket:  Use Rest API
### Save 
```
curl -s -vvv -X POST "http://localhost:8080/jsie" -H 'content-type: application/json' -d '{ "value": "hello trendev" }' | jq
```
### Find
```
curl -s -vvv "http://localhost:8080/jsie" -H 'content-type: application/json' | jq
```
