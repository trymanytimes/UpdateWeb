#!/bin/bash

set -e

VERSION=$1
UIVERSION=${2:-$1}

if [[ -z $VERSION ]]
then
cat <<EOF

<------------------------------------------------>
    Usage:
        ./gen_image.sh {version}
        ./gen_image.sh {ddi-controller version} {ddi-web version}
<------------------------------------------------>

EOF
    exit 1
fi

cat <<EOF

<------------------------------------------------>
    Building ...
<------------------------------------------------>

EOF

cat <<'EOF' | docker build -f - -t linkingthing/ddi-controller:${VERSION}-${UIVERSION} --build-arg version=${VERSION} --build-arg uiversion=${UIVERSION} .
ARG version
ARG uiversion

FROM linkingthing/ddi-controller:$version as go
FROM linkingthing/ddi-web:$uiversion as js

FROM alpine:3.12

COPY --from=go /ddi-controller /
COPY --from=js /opt/website /opt/website

ENTRYPOINT ["/ddi-controller"]
EOF

if [[ $? -eq 0 ]]
then
docker image prune -f
cat <<EOF

<------------------------------------------------>
  Image build complete.
  Build: zdnscloud/ddi-controller:${VERSION}-${UIVERSION}
<------------------------------------------------>

EOF
else
cat <<EOF

<------------------------------------------------>
  Image build failure.
<------------------------------------------------>

EOF
fi
