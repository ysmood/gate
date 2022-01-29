FROM golang as go

RUN mkdir /app
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build ./cmd/server

FROM alpine

WORKDIR /root

COPY --from=go /app/server .

CMD ./server
