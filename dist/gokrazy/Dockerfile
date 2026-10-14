# Builds a statically linked, non-SUID apptainer installation and packs it
# into a gokrazy extrafiles tarball. Use dist/gokrazy/build.sh rather than
# invoking this directly.

ARG GOLANG_VERSION="pass-in-as-build-arg"

FROM golang:${GOLANG_VERSION}-alpine AS build

ARG VERSION=dev
ARG PREFIX=/usr/local/apptainer
ARG SQUASHFS_TOOLS_VERSION=4.7.2
ARG SQUASHFS_TOOLS_SHA256=4672b5c47d9418d3a5ae5b243defc6d9eae8275b9771022247c6a6082c815914
ARG SHADOW_VERSION=4.20.3
ARG SHADOW_SHA256=623be9429f64fa2165bfc5957c6a642ec309c5cf23c0d53855c99f7591922297

RUN apk add -q --no-cache bash alpine-sdk linux-headers gawk sed curl file xz \
    zlib-dev zlib-static xz-dev xz-static zstd-dev zstd-static \
    lz4-dev lz4-static

# squashfs-tools, so that "apptainer pull/build" can create SIF images
RUN curl -fsSL -o /tmp/sqfs.tar.gz \
    "https://github.com/plougher/squashfs-tools/archive/refs/tags/${SQUASHFS_TOOLS_VERSION}.tar.gz" && \
    echo "${SQUASHFS_TOOLS_SHA256}  /tmp/sqfs.tar.gz" | sha256sum -c - && \
    mkdir /tmp/sqfs && tar xzf /tmp/sqfs.tar.gz -C /tmp/sqfs --strip-components=1 && \
    make -C /tmp/sqfs/squashfs-tools -j"$(nproc)" LZO_SUPPORT=0 \
    EXTRA_LDFLAGS=-static mksquashfs unsquashfs

# newuidmap/newgidmap from shadow, required by --fakeroot in a non-SUID
# installation. --without-nscd is not used as it breaks the shadow build.
RUN curl -fsSL -o /tmp/shadow.tar.xz \
    "https://github.com/shadow-maint/shadow/releases/download/${SHADOW_VERSION}/shadow-${SHADOW_VERSION}.tar.xz" && \
    echo "${SHADOW_SHA256}  /tmp/shadow.tar.xz" | sha256sum -c - && \
    mkdir /tmp/shadow && tar xJf /tmp/shadow.tar.xz -C /tmp/shadow --strip-components=1 && \
    cd /tmp/shadow && \
    ./configure -q --disable-shared --enable-static --disable-nls --disable-man \
    --disable-logind --without-libpam --without-audit --without-selinux \
    --without-acl --without-tcb --without-btrfs --without-sssd --without-libbsd \
    --with-yescrypt=no --with-bcrypt=no && \
    make -C lib -j"$(nproc)" && \
    make -C src newuidmap newgidmap LDFLAGS=-all-static

WORKDIR /build
COPY . /build

RUN echo "${VERSION}" > VERSION && \
    ./mconfig -s -S --without-suid --without-seccomp --without-libsubid \
    --without-network -b /builddir -p "${PREFIX}" && \
    make -C /builddir -j"$(nproc)" && \
    make -C /builddir install DESTDIR=/out

# gokrazy turns zero-length extra files into directories, so give
# capability.json valid content and drop other empty files (all optional).
# gokrazy has no /etc/passwd or /etc/group, which the starter requires, nor
# /etc/subuid and /etc/subgid, which --fakeroot requires. "build --fakeroot"
# also needs /root to exist, to bind the home directory on it.
RUN set -e; \
    root="/out${PREFIX}"; \
    install -m 0755 /tmp/sqfs/squashfs-tools/mksquashfs \
    /tmp/sqfs/squashfs-tools/unsquashfs /tmp/shadow/src/newuidmap \
    /tmp/shadow/src/newgidmap "${root}/libexec/apptainer/bin/"; \
    rm -rf "${root}/share" "${root}/libexec/apptainer/lib" \
    "${root}/bin/singularity" "${root}/bin/run-singularity"; \
    find "${root}" -type f -perm -u+x -exec sh -c \
    'file "$1" | grep -Eq "ELF.*(statically|static-pie) linked" && strip "$1"' _ {} \; ; \
    for f in "${root}/bin/apptainer" "${root}/libexec/apptainer/bin/"*; do \
    if file "$f" | grep -q "ELF" && ! file "$f" | grep -Eq "(statically|static-pie) linked"; then \
    echo "error: $f is not statically linked" >&2; exit 1; fi; done; \
    echo '{}' > "${root}/etc/apptainer/capability.json"; \
    find /out -type f -size 0 -delete; \
    mkdir -p /out/etc; \
    printf 'root:x:0:0:root:/perm/home:/bin/sh\n' > /out/etc/passwd; \
    printf 'root:x:0:\ntty:x:5:\n' > /out/etc/group; \
    printf 'root:100000:65536\n' > /out/etc/subuid; \
    printf 'root:100000:65536\n' > /out/etc/subgid; \
    install -d -m 0700 /out/root; \
    tar --numeric-owner --owner=0 --group=0 -C /out -cf /extrafiles.tar etc root usr

FROM scratch
COPY --from=build /extrafiles.tar /
