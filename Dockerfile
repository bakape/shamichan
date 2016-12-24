# Used only for testing
FROM bakape/meguca
MAINTAINER bakape <scorpid33@gmail.com>
EXPOSE 8000
ENV PATH="${PATH}:/usr/local/go/bin:/root/.cargo/bin"
RUN mkdir -p /meguca
WORKDIR /meguca
ENTRYPOINT service rethinkdb start && make test

RUN apt-get update && apt-get dist-upgrade -y
COPY . .
RUN make all
