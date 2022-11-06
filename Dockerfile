FROM library/golang:1.19.2-bullseye

WORKDIR /app

RUN useradd lowpriv

# ref https://github.com/vertexproject/vtx-base-image/blob/master/python310/Dockerfile#L17
RUN set -ex \
    && apt-get clean \
    && apt-get update \
    && apt-get -y upgrade \
    && apt-get install -y curl \
    && apt remove -y build-essential \
    && apt-get clean && apt-get purge \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /go/chaldeploy .

USER lowpriv

EXPOSE 5050
ENTRYPOINT [ "/go/chaldeploy" ]

HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 CMD curl -k http://localhost:5050/healthcheck || exit 1