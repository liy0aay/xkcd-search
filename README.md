## About

Service for searching XKCD comics and retrieving their explanations.

The project consists of several backend services communicating over gRPC and an API gateway
that exposes HTTP endpoints for clients.

## Functionality

- Search XKCD comics by keywords
- Indexed search for faster results
- Retrieve comic explanations via explainxkcd
- Text normalization for better search quality
- Event-based updates using NATS

## Architecture

- Hexagonal (ports & adapters) architecture
- API Gateway for HTTP access
- Independent backend services
- gRPC for internal communication
- NATS as a message broker

## Tech Stack

- Go
- gRPC
- NATS
- PostgreSQL

## Running

```bash
docker compose up --build