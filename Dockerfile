ARG ruby_version=3.3
ARG base_image=ghcr.io/alphagov/govuk-ruby-base:$ruby_version
ARG builder_image=ghcr.io/alphagov/govuk-ruby-builder:$ruby_version
ARG TARGETPLATFORM=linux/arm64
ARG APP_HOME=/app


FROM --platform=$TARGETPLATFORM $builder_image AS builder
WORKDIR $APP_HOME
COPY Gemfile* .ruby-version ./
RUN bundle install
RUN export BUNDLE_PATH=$(which bundle);echo $BUNDLE_PATH;
COPY . .

FROM --platform=$TARGETPLATFORM $base_image
WORKDIR $APP_HOME
COPY --from=builder $BUNDLE_PATH $BUNDLE_PATH
COPY --from=builder $APP_HOME .

RUN apt-get update && \
    apt-get install -y git ssh

RUN mkdir -p ~/.ssh && \
    ssh-keyscan github.com >> ~/.ssh/known_hosts

# Give app user ownership to allow git to make changes
RUN chown -R app:app /app

USER app
ENV ENV_MESSAGE_EXAMPLE="Environment message from example user"
EXPOSE 3000 9394
CMD rackup -o 0.0.0.0 -p 3000
