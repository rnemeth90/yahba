# ── Stage 1: compile ngx_brotli and headers-more dynamic modules ──────────────
FROM nginx:stable-alpine AS builder

RUN apk add --no-cache \
    gcc \
    g++ \
    libc-dev \
    make \
    openssl-dev \
    pcre2-dev \
    zlib-dev \
    linux-headers \
    git \
    cmake

# Download nginx source at the exact version of the installed binary so the
# compiled .so files are ABI-compatible.
RUN set -ex && \
    NGINX_VER=$(nginx -v 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+') && \
    echo "nginx version: $NGINX_VER" && \
    wget -q "http://nginx.org/download/nginx-${NGINX_VER}.tar.gz" && \
    tar -xzf "nginx-${NGINX_VER}.tar.gz" && \
    mv "nginx-${NGINX_VER}" /nginx-src

# ngx_brotli: --recurse-submodules pulls the brotli C library
RUN git clone --depth=1 --recurse-submodules \
        https://github.com/google/ngx_brotli.git /ngx_brotli

# Build the brotli C library first; ngx_brotli links against it statically.
# The symlink step handles cmake naming the output libbrotlienc-static.a on
# some platforms instead of libbrotlienc.a.
RUN cd /ngx_brotli/deps/brotli && \
    mkdir -p out && cd out && \
    cmake -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF .. && \
    make -j$(nproc) && \
    for lib in brotlicommon brotlienc; do \
        [ -f lib${lib}.a ] || ln -sf lib${lib}-static.a lib${lib}.a; \
    done

RUN git clone --depth=1 \
        https://github.com/openresty/headers-more-nginx-module.git /headers-more

RUN cd /nginx-src && \
    ./configure \
        --with-compat \
        --add-dynamic-module=/ngx_brotli \
        --add-dynamic-module=/headers-more && \
    make modules

# ── Stage 2: runtime image ────────────────────────────────────────────────────
FROM nginx:stable-alpine

# nginx-module-njs is published to the same nginx.org Alpine repo used by this
# base image, so the version is guaranteed to match the installed nginx binary.
RUN apk add --no-cache nginx-module-njs

# Create the module directory path used in the production nginx.conf
RUN mkdir -p /var/nginx/modules

COPY --from=builder /nginx-src/objs/ngx_http_brotli_filter_module.so       /var/nginx/modules/
COPY --from=builder /nginx-src/objs/ngx_http_brotli_static_module.so       /var/nginx/modules/
COPY --from=builder /nginx-src/objs/ngx_http_headers_more_filter_module.so /var/nginx/modules/

# nginx-module-njs installs to /usr/lib/nginx/modules/; copy to the path the
# production config expects.
RUN cp /usr/lib/nginx/modules/ngx_http_js_module.so /var/nginx/modules/
