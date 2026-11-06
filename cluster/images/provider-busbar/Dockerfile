ARG TERRAFORM_PROVIDER_GO_VERSION=1.27.3
FROM --platform=$BUILDPLATFORM golang:${TERRAFORM_PROVIDER_GO_VERSION}-alpine AS terraform-provider-build

ARG TERRAFORM_PROVIDER_GO_VERSION
ARG TARGETOS
ARG TARGETARCH
ARG TERRAFORM_PROVIDER_REPO
ARG TERRAFORM_PROVIDER_REF
ARG TERRAFORM_PROVIDER_VERSION

RUN apk --no-cache add ca-certificates git
COPY --chmod=0755 build_terraform_provider.sh /usr/local/bin/build_terraform_provider.sh
RUN GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
    /usr/local/bin/build_terraform_provider.sh /src /out/terraform-provider-busbar

FROM alpine:3.23.4
RUN apk --no-cache add ca-certificates bash

ARG TARGETOS
ARG TARGETARCH

ADD "bin/${TARGETOS}_${TARGETARCH}/provider" /usr/local/bin/provider

ENV USER_ID=65532

# Setup Terraform environment

## Provider-dependent configuration
ARG TERRAFORM_VERSION
ARG TERRAFORM_PROVIDER_SOURCE
ARG TERRAFORM_PROVIDER_REF
ARG TERRAFORM_PROVIDER_VERSION
ARG TERRAFORM_PROVIDER_DOWNLOAD_NAME
ARG TERRAFORM_NATIVE_PROVIDER_BINARY
ARG TERRAFORM_PROVIDER_DOWNLOAD_URL_PREFIX

## End of - Provider-dependent configuration

LABEL org.getbusbar.terraform-provider.revision=${TERRAFORM_PROVIDER_REF}

ENV PLUGIN_DIR=/terraform/provider-mirror/registry.terraform.io/${TERRAFORM_PROVIDER_SOURCE}/${TERRAFORM_PROVIDER_VERSION}/${TARGETOS}_${TARGETARCH}
ENV TF_CLI_CONFIG_FILE=/terraform/.terraformrc
ENV TF_FORK=0

RUN mkdir -p ${PLUGIN_DIR}

ADD https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_${TARGETOS}_${TARGETARCH}.zip /tmp
COPY --from=terraform-provider-build --chmod=0755 /out/terraform-provider-busbar ${PLUGIN_DIR}/${TERRAFORM_NATIVE_PROVIDER_BINARY}
ADD terraformrc.hcl ${TF_CLI_CONFIG_FILE}

RUN unzip /tmp/terraform_${TERRAFORM_VERSION}_${TARGETOS}_${TARGETARCH}.zip -d /usr/local/bin \
  && chmod +x /usr/local/bin/terraform \
  && rm /tmp/terraform_${TERRAFORM_VERSION}_${TARGETOS}_${TARGETARCH}.zip \
  && chown -R ${USER_ID}:${USER_ID} /terraform
# End of - Setup Terraform environment

# Provider controller needs these environment variable at runtime
ENV TERRAFORM_VERSION=${TERRAFORM_VERSION}
ENV TERRAFORM_PROVIDER_SOURCE=${TERRAFORM_PROVIDER_SOURCE}
ENV TERRAFORM_PROVIDER_VERSION=${TERRAFORM_PROVIDER_VERSION}
ENV TERRAFORM_NATIVE_PROVIDER_PATH=${PLUGIN_DIR}/${TERRAFORM_NATIVE_PROVIDER_BINARY}

USER ${USER_ID}
EXPOSE 8080

ENTRYPOINT ["provider"]
