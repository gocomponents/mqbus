FROM alpine:latest
ADD zoneinfo.tar.gz /
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
MAINTAINER jinyidong jinyidong@outlook.com
ADD mqbus /
CMD ["/mqbus"]