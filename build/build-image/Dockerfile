
# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:GO_VERSION

# Avoid permission issues when glide pulls mercurial repos as root
RUN printf "[trusted]\nusers = *\n" > /root/.hgrc

# Install glide as root
ENV GLIDE_VERSION=v0.12.3 \
    GLIDE_HOME=/go/src/github.com/openshift/cluster-operator/.glide
RUN curl -sSL https://github.com/Masterminds/glide/releases/download/$GLIDE_VERSION/glide-$GLIDE_VERSION-linux-amd64.tar.gz \
    | tar -vxz -C /usr/local/bin --strip=1
RUN go get -u github.com/sgotti/glide-vc

# Install the golint, use this to check our source for niceness
RUN go get -u github.com/golang/lint/golint

# Create the full dir tree that we'll mount our src into when we run the image
#RUN mkdir -p /go/src/github.com/openshift/cluster-operator
RUN mkdir -p /go/src/github.com/enxebre/cluster-api-provider-aws

# Default to our src dir
WORKDIR /go/src/github.com/enxebre/cluster-api-provider-aws
