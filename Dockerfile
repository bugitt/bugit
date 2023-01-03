FROM ubuntu

RUN apt update && apt install build-essential git -y

RUN git config --global init.defaultBranch main

WORKDIR /root/

EXPOSE 22 3000

COPY bugit ./
