FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG GIT_COMMIT=unknown
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.gitCommit=${GIT_COMMIT}" -o /redir .

FROM gcr.io/distroless/static-debian12
COPY --from=build /redir /redir
EXPOSE 4000
USER 1000:1000
ENTRYPOINT ["/redir"]
