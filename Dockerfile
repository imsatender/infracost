FROM golang:1.15 as builder

ARG ARCH=linux
ARG DEFAULT_TERRAFORM_VERSION=0.14.6
ARG TERRAGRUNT_VERSION=0.28.5
# Use infracost-usage.yml instead of the provider, see https://www.infracost.io/docs/usage_based_resources
ARG TERRAFORM_PROVIDER_INFRACOST_VERSION=latest

# Set Environment Variables
SHELL ["/bin/bash", "-c"]
ENV HOME /app
ENV CGO_ENABLED 0

# Install Packages
RUN apt-get update -q && apt-get -y install zip jq -y

# Install latest of each Terraform version after 0.12 as we don't support versions before that
RUN AVAILABLE_TERRAFORM_VERSIONS="0.12.30 0.13.6 ${DEFAULT_TERRAFORM_VERSION}" && \
    for VERSION in ${AVAILABLE_TERRAFORM_VERSIONS}; do \
    wget -q https://releases.hashicorp.com/terraform/${VERSION}/terraform_${VERSION}_linux_amd64.zip && \
    wget -q https://releases.hashicorp.com/terraform/${VERSION}/terraform_${VERSION}_SHA256SUMS && \
    sed -n "/terraform_${VERSION}_linux_amd64.zip/p" terraform_${VERSION}_SHA256SUMS | sha256sum -c && \
    unzip terraform_${VERSION}_linux_amd64.zip -d /tmp && \
    mv /tmp/terraform /usr/bin/terraform_${VERSION} && \
    chmod +x /usr/bin/terraform_${VERSION} && \
    rm terraform_${VERSION}_linux_amd64.zip && \
    rm terraform_${VERSION}_SHA256SUMS; \
    done && \
    ln -s /usr/bin/terraform_0.12.30 /usr/bin/terraform_0.12 && \
    ln -s /usr/bin/terraform_0.13.6 /usr/bin/terraform_0.13 && \
    ln -s /usr/bin/terraform_${DEFAULT_TERRAFORM_VERSION} /usr/bin/terraform_0.14 && \
    ln -s /usr/bin/terraform_${DEFAULT_TERRAFORM_VERSION} /usr/bin/terraform

# Install Terragrunt
RUN wget -q https://github.com/gruntwork-io/terragrunt/releases/download/v$TERRAGRUNT_VERSION/terragrunt_linux_amd64
RUN mv terragrunt_linux_amd64 /usr/bin/terragrunt && \
    chmod +x /usr/bin/terragrunt

WORKDIR /app
COPY scripts/install_provider.sh scripts/install_provider.sh
RUN scripts/install_provider.sh ${TERRAFORM_PROVIDER_INFRACOST_VERSION} /usr/bin/

# Build Application
COPY . .
RUN make deps
RUN NO_DIRTY=true make build

# Application
FROM alpine:3.13 as app
# Tools needed for running diffs in CI integrations
RUN apk --update --no-cache add ca-certificates openssl openssh-client curl git jq
WORKDIR /root/
# Scripts are used by CI integrations and other use-cases
COPY scripts /scripts
COPY --from=builder /usr/bin/terraform* /usr/bin/
COPY --from=builder /usr/bin/terragrunt /usr/bin/
COPY --from=builder /app/build/infracost /usr/bin/
RUN chmod +x /usr/bin/infracost

ENTRYPOINT [ "infracost" ]
