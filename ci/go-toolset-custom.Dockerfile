FROM registry.redhat.io/rhel8/go-toolset:1.13

USER 0

RUN dnf install -y diffutils && \
    dnf clean all

USER 1001
