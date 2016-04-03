FROM golang:1.5.3-onbuild

# running golang linter to find problems that the compliter did not find
RUN go vet ./...

# running any test that exist in the project
RUN go test ./...