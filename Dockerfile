FROM golang:1.16.4-alpine as build

WORKDIR /master-broker
COPY go.mod ./
#COPY go.sum ./
RUN go mod tidy
RUN go mod vendor
COPY ./ ./
RUN go build -o ./built/project


FROM alpine
FROM alpine
ENV GRPC_PORT=4000
ENV PROMETHEUS_PORT=5000
COPY --from=build /master-broker/built/project ./master_broker
EXPOSE $GRPC_PORT
EXPOSE $PROMETHEUS_PORT
CMD [ "./master_broker" ]
