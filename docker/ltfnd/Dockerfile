FROM golang:1.12-alpine as builder

MAINTAINER Olaoluwa Osuntokun <lightning.engineering>

# Grab and install the latest version of roasbeef's fork of ltfnd and all
# related dependencies.
WORKDIR $GOPATH/src/github.com/litecoinfinance/ltfnd
RUN apk add --no-cache --update alpine-sdk git
RUN git clone https://github.com/litecoinfinance/ltfnd ./
RUN GO111MODULE=on go install -v . ./cmd/...
RUN GO111MODULE=on go install . ./cmd/ltfnctl ./cmd/gencerts

# Start a new image
FROM alpine as final

# Expose mainnet ports (server, rpc)
EXPOSE 8333 8334

# Expose testnet ports (server, rpc)
EXPOSE 18333 18334

# Expose simnet ports (server, rpc)
EXPOSE 18555 18556

# Expose segnet ports (server, rpc)
EXPOSE 28901 28902

# Copy the compiled binaries from the builder image.
COPY --from=builder /go/bin/ltfnctl /bin/
COPY --from=builder /go/bin/ltfnd /bin/
COPY --from=builder /go/bin/gencerts /bin/

COPY "start-ltfnctl.sh" .
COPY "start-ltfnd.sh" .

RUN apk add --no-cache \
    bash \
    ca-certificates \
&&  chmod +x start-ltfnctl.sh \
&&  chmod +x start-ltfnd.sh \
&&  mkdir "/rpc" "/root/.ltfnd" "/root/.ltfnctl" \
&&  touch "/root/.ltfnd/ltfnd.conf" \
# "ltfnctl" and "lnd" to "ltfnd" over docker links.
&&  "/bin/gencerts" --host="*" --directory="/rpc" --force

# Create a volume to house pregenerated RPC credentials. This will be
# shared with any lnd, btcctl containers so they can securely query ltfnd's RPC
# server.
# You should NOT do this before certificate generation!
# Otherwise manually generated certificate will be overridden with shared
# mounted volume! For more info read dockerfile "VOLUME" documentation.
VOLUME ["/rpc"]
