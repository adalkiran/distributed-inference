# See for available variants: https://hub.docker.com/_/redis?tab=tags
ARG VARIANT=7.0.1-bullseye
FROM redis:${VARIANT}

COPY entrypoint.sh /

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install gettext-base \
    && chmod +x /entrypoint*.sh

ENTRYPOINT "/entrypoint.sh"