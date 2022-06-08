FROM registry.ci.openshift.org/openshift/release:golang-1.18 as builder

WORKDIR /hypershift

COPY . .

RUN make build

FROM quay.io/openshift/origin-base:4.10
COPY --from=builder /hypershift/bin/hypershift \
                    /hypershift/bin/hypershift-operator \
                    /hypershift/bin/control-plane-operator \
     /usr/bin/

# Copying these files so commands can be invoked consistently via Make.
COPY --from=builder /hypershift/Makefile /Makefile
COPY --from=builder /hypershift/hack/ci-test-e2e.sh /hack/ci-test-e2e.sh

RUN cd /usr/bin && \
    ln -s control-plane-operator ignition-server && \
    ln -s control-plane-operator konnectivity-socks5-proxy && \
    ln -s control-plane-operator availability-prober && \
    ln -s control-plane-operator token-minter

ENTRYPOINT /usr/bin/hypershift

LABEL io.openshift.hypershift.control-plane-operator-subcommands=true
LABEL io.openshift.hypershift.control-plane-operator-skips-haproxy=true
LABEL io.openshift.hypershift.ignition-server-healthz-handler=true
