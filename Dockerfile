# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

# Copy the local package files to the container's workspace.
ADD . /Users/apple/go/src/github.com/LunchBadger/git-api

# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
RUN dep

# Run the outyet command by default when the container starts.
ENTRYPOINT /go/bin/git-api

# Document that the service listens on port 8080.
EXPOSE 8080