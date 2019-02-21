# syntax = docker/dockerfile:experimental
FROM python:3

RUN pip install awscli
# --mount=type=secret,id=aws,target=/root/.aws/credentials
RUN --mount=type=secret,id=aws,target=/root/.aws/credentials aws s3 cp s3://demo-docker-madrid-20-03-2019/message message
