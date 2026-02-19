module github.com/gavinadlan/tripnest/backend/search-service

go 1.24.0

require (
	github.com/gavinadlan/tripnest/backend/common v0.0.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/joho/godotenv v1.5.1
	go.mongodb.org/mongo-driver v1.11.2
	github.com/go-chi/chi/v5 v5.2.5
)

replace github.com/gavinadlan/tripnest/backend/common => ../common
