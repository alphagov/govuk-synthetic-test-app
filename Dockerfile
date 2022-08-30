FROM ruby:3.0.4

WORKDIR /app

COPY ./ /app

ENV ENV_MESSAGE_EXAMPLE="Environment message from example user"

RUN bundle install

EXPOSE 9292 9394

CMD rackup -o 0.0.0.0
