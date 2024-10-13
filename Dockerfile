# build go app with golang image
FROM golang:1.23 AS builder
COPY backend/*.go backend/go.mod backend/go.sum /app/
WORKDIR /app
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# build frontend with node image
FROM node:20 AS frontend
COPY frontend /tmp/frontend
WORKDIR /tmp/frontend
RUN yarn install
RUN yarn build

# move app into minimal executable image
FROM alpine:latest AS runner
RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /app
COPY --from=builder /app/app /app/
COPY --from=frontend /tmp/frontend/dist /app/dist
EXPOSE 8080
CMD ["./app"]
