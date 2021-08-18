#!/bin/bash
#
# Copyright 2020 IBM Corporation.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

LOG_INDEX=$1

UUID=$(curl -s "https://rekor.sigstore.dev/api/v1/log/entries/?logIndex=${LOG_INDEX}" | jq keys | jq -c '.[]')

if [ -z "$UUID" ]; then
    echo "Please wait few secs before querying sigstore log"
    exit 0
fi

echo $UUID
QUERY=".${UUID}.attestation.data"
echo $QUERY
sleep 2
curl -s "https://rekor.sigstore.dev/api/v1/log/entries/?logIndex=${LOG_INDEX}" | jq -r $QUERY | base64 -D | base64 -D | jq .





