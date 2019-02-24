####################################################
# GOLANG BUILDER
####################################################
FROM golang:1.11 as go_builder

COPY . /go/src/github.com/malice-plugins/sophos
WORKDIR /go/src/github.com/malice-plugins/sophos
RUN go get -u github.com/golang/dep/cmd/dep && dep ensure
RUN go build -ldflags "-s -w -X main.Version=v$(cat VERSION) -X main.BuildTime=$(date -u +%Y%m%d)" -o /bin/avscan

####################################################
# PLUGIN BUILDER
####################################################
FROM ubuntu:bionic

LABEL maintainer "https://github.com/blacktop"

LABEL malice.plugin.repository = "https://github.com/malice-plugins/sophos.git"
LABEL malice.plugin.category="av"
LABEL malice.plugin.mime="*"
LABEL malice.plugin.docker.engine="*"

# Create a malice user and group first so the IDs get set the same way, even as
# the rest of this may change over time.
RUN groupadd -r malice \
  && useradd --no-log-init -r -g malice malice \
  && mkdir /malware \
  && chown -R malice:malice /malware

# Install Requirements
RUN buildDeps='wget ca-certificates' \
  && DEBIAN_FRONTEND=noninteractive apt-get update -qq \
  && apt-get install -yq $buildDeps \
  && echo "===> Install Sophos..." \
  && cd /tmp \
  && wget -q https://github.com/maliceio/malice-av/raw/master/sophos/sav-linux-free-9.tgz \
  && tar xzvf sav-linux-free-9.tgz \
  && ./sophos-av/install.sh /opt/sophos --update-free --acceptlicence --autostart=False --enableOnBoot=False --automatic --ignore-existing-installation --update-source-type=s \
  && echo "===> Update Sophos..." \
  && mkdir -p /opt/malice \
  && /opt/sophos/update/savupdate.sh \
  && /opt/sophos/bin/savconfig set DisableFeedback true \
  && echo "===> Clean up unnecessary files..." \
  && apt-get purge -y --auto-remove $buildDeps \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* /root/.gnupg

# Ensure ca-certificates is installed for elasticsearch to use https
RUN apt-get update -qq && apt-get install -yq --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY --from=go_builder /bin/avscan /bin/avscan

# Add EICAR Test Virus File to malware folder
ADD http://www.eicar.org/download/eicar.com.txt /malware/EICAR

WORKDIR /malware

ENTRYPOINT ["/bin/avscan"]
CMD ["--help"]

####################################################
####################################################
# https://downloads.sophos.com/inst/98Foy3KvwUcDhjL4YwnVAwZD01NzM0/sav-linux-free-9.tgz
