# syntax=docker/dockerfile:1

# Mailboxsweep Web Frontend
# This site is designed to be served as static HTML + JS + JSON behind a CDN
# This docker image is to facilitate local penetration testing against a production
# build without running any code on bare metal

# MUST run in the context of the root of the repository
# so the .yml files from backend (symlinks in local environment) can be copied

FROM node:20.9

WORKDIR /opt/build

COPY web/package.json web/package-lock.json ./

RUN npm install

COPY web ./
COPY backend/payment/plans.yml ./
COPY backend/inboxuser/agreements/*.yml ./agreements/
COPY openapi.yml ../

# RUN npm run openapi
RUN npm run build:docker
COPY web/serve.json ./out

EXPOSE 3000

CMD ["npm", "run", "serve"]