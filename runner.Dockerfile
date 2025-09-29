FROM golang:1.25.1-alpine3.22

RUN apk add --no-cache aws-cli

ARG USER=app
ENV HOME=/home/$USER

# install sudo as root
RUN apk add --update sudo

# add new user
# RUN adduser -D $USER \
#         && echo "$USER ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/$USER \
#         && chmod 0440 /etc/sudoers.d/$USER

RUN adduser -D $USER \
        && mkdir -p /etc/sudoers.d \
        && echo "$USER ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/$USER \
        && chmod 0440 /etc/sudoers.d/$USER

# USER appuser
# WORKDIR /app

USER $USER
WORKDIR $HOME

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# CMD [ "/assume_role" ]

CMD ["sleep", "6000"]
