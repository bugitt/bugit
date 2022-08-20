FROM bitnami/git:2-debian-11

RUN git config --global init.defaultBranch main

RUN apt update && apt install build-essential -y

WORKDIR /root/

EXPOSE 22 3000

COPY bugit ./
