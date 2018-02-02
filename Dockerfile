FROM bakape/meguca
EXPOSE 8000
ENV PATH="${PATH}:/usr/local/go/bin"
RUN mkdir -p /meguca
WORKDIR /meguca
COPY . .
RUN make all
