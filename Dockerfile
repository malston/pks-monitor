FROM golang:latest

ENV APP_NAME pks-monitor
ENV PORT 8080

COPY . /go/src/${APP_NAME}
WORKDIR /go/src/${APP_NAME}

# RUN go get ./
RUN go build -o ${APP_NAME} -mod=vendor ./cmd/main.go

CMD ./${APP_NAME}

EXPOSE ${PORT}
