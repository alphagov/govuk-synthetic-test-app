FROM golang:1.25.1-alpine3.22

RUN apk add --no-cache aws-cli

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN mkdir /app
RUN chown appuser:appgroup /app
COPY --chown=appuser:appgroup . /app

WORKDIR /app

USER appuser

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# CMD [ "/assume_role" ]

CMD ["sleep", "6000"]
