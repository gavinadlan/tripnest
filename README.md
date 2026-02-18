# TripNest: Distributed Travel Booking System

TripNest is a distributed system designed to handle high-concurrency travel bookings with a resilient, event-driven architecture.

## Architecture Highlights
- **Microservices**: Decoupled services with single responsibility.
- **Event-Driven**: Kafka for asynchronous communication and choreography-based Sagas.
- **Polyglot Persistence**: PostgreSQL for transactional data, MongoDB for search/analytics.
- **Scalability**: Stateless services, horizontal scaling, caching with Redis.
- **Frontend**: Minimal Next.js UI using REST APIs.

## Tech Stack
- **Backend**: Go (Golang)
- **Database**: PostgreSQL, MongoDB, Redis
- **Messaging**: Kafka
- **Infrastructure**: Docker, Docker Compose
- **Frontend**: Next.js, TailwindCSS

## Service Breakdown

| Service | Status | Tech | Responsibility |
| :--- | :--- | :--- | :--- |
| **User Service** | ✅ Done | Go, PostgreSQL | User management, Authentication (JWT), Clean Architecture. |
| **Search Service** | 🚧 Pending | Go, MongoDB, Redis | High-performance flight/hotel search. |
| **Booking Service** | ✅ Done | Go, PostgreSQL, Kafka | Order management, Booking state machine. |
| **Payment Service** | 🚧 Pending | Go, PostgreSQL, Kafka | Payment processing, Idempotency. |
| **Notification Service** | 🚧 Pending | Go, Kafka Consumer | Async notifications (Email/SMS). |
| **Recommendation Service** | 🚧 Pending | Go, MongoDB | Personalized travel suggestions. |
| **API Gateway** | 🚧 Pending | Nginx / Go | Entry point, Rate limiting. |

## Event Choreography (Saga Pattern)
We use a decentralized choreography-based Saga to manage distributed transactions.
1.  **Booking Created** -> `booking.created`
2.  **Payment Processed** -> `payment.success` / `payment.failed`
3.  **Booking Completed** -> `booking.confirmed` / `booking.cancelled`
4.  **Notification Sent** -> `notification.sent`

## Setup & Running
*(Instructions to run with Docker Compose will be added here)*

## Project Structure
This is a monorepo containing:
- `backend/`: Go microservices using Go Workspaces.
- `frontend/`: Next.js frontend application.
