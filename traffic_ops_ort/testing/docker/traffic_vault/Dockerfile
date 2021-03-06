# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
############################################################
# Dockerfile to build Riak container images
#   as Traffic Vault for Traffic Control 1.6.0
# Based on CentOS 6.6
############################################################

FROM basho/riak-kv:ubuntu-2.2.3

EXPOSE 8087 8088 8098

RUN rm -rfv /etc/riak/prestart.d/* /etc/riak/poststart.d/*

RUN echo 'APT::Install-Recommends 0;' >> /etc/apt/apt.conf.d/01norecommends \
 && echo 'APT::Install-Suggests 0;' >> /etc/apt/apt.conf.d/01norecommends \
 && rm /etc/apt/sources.list.d/basho_riak.list \
 && apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y net-tools ca-certificates dnsutils gettext-base \
 && rm -rf /var/lib/apt/lists/* && rm -rf /etc/apt/apt.conf.d/docker-gzip-indexes

ADD traffic_vault/prestart.d/* /etc/riak/prestart.d/
ADD traffic_vault/poststart.d/* /etc/riak/poststart.d/
ADD traffic_vault/run.sh traffic_vault/sslkeys.xml /

CMD /run.sh
