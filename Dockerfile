FROM ruby:3.0.4

WORKDIR /app

COPY ./ /app

ENV HELM_MESSAGE="from dockerfile"

RUN bundle install

EXPOSE 9292 9394

CMD rackup -o 0.0.0.0
