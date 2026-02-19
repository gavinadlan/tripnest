module github.com/gavinadlan/tripnest/backend/search-service

go 1.23.4

require (
	github.com/gavinadlan/tripnest/backend/common v0.0.0
	github.com/go-chi/chi/v5 v5.2.5
	github.com/go-redis/redis/v8 v8.11.5
	github.com/joho/godotenv v1.5.1
	go.mongodb.org/mongo-driver v1.11.2
)

require github.com/go-chi/cors v1.2.2 // indirect

replace github.com/gavinadlan/tripnest/backend/common => ../common
