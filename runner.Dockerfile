FROM golang:1.25.1-alpine3.22

RUN apk add --no-cache aws-cli curl

ARG USER=app
ENV HOME=/home/$USER

RUN addgroup -g 1000 $USER \
    && adduser -u 1000 -G $USER -D $USER \
    && chown -R $USER:$USER $HOME \
    && chown $USER:$USER /tmp

USER $USER
WORKDIR $HOME

COPY go.mod .
COPY go.sum .

RUN go mod download
RUN go install github.com/onsi/ginkgo/v2/ginkgo

COPY . .

# CMD ["sleep", "60000"]

CMD ["ginkgo", "helpers"]