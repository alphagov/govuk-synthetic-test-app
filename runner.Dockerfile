FROM golang:1.25.1-alpine3.22

RUN apk add --no-cache aws-cli
RUN mkdir -p /app/.aws

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go install assume_role

# CMD [ "/assume_role" ]

CMD ["sleep", "6000"]
