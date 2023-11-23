FROM rockylinux:9

LABEL maintainer="Edo Shor <edoshor@gmail.com>"
LABEL description="Provides an image with BB fork of Janus Gateway for Galaxy 3"

# docker build arguments
ARG BUILD_SRC="/usr/local/src"

RUN dnf install -y epel-release \
    && dnf upgrade -y
RUN dnf install -y \
    yum-utils \
    libconfig \
    libmicrohttpd \
    openssl \
    glib2 \
    opus \
    libogg \
    pkgconfig \
    libtool \
    autoconf \
    automake \
    cmake \
    cpp \
    gcc-c++ \
    gcc \
    git \
    make \
    wget \
    golang \
    glib2-devel \
    libcurl-devel \
    python3-pip \
    tuned \
    dnf-plugins-core \
    unzip \
    pip \
    xz \
    && dnf --enablerepo=devel install -y \
    jansson-devel \
    libconfig-devel \
    libmicrohttpd-devel \
    meson \
    opus-devel \
    libwebsockets-devel \
    speexdsp-devel \
    texinfo
RUN pip3 install ninja meson
RUN tuned-adm profile throughput-performance


RUN /usr/sbin/groupadd -r janus \
    && /usr/sbin/useradd -r -g janus janus \
    && mkdir -p $BUILD_SRC


### getopt
RUN cd $BUILD_SRC \
    && curl -qL http://ftp.gnu.org/gnu/gengetopt/gengetopt-2.23.tar.xz -o gengetopt-2.23.tar.xz \
    && tar xfv gengetopt-2.23.tar.xz \
    && cd gengetopt-2.23 \
    && ./configure --prefix=/usr --libdir=/usr/lib64 \
    && make \
    && make install

### nice
RUN cd $BUILD_SRC \
    && curl -qL https://gitlab.freedesktop.org/libnice/libnice/-/archive/0.1.18/libnice-0.1.18.tar.gz -o libnice-0.1.18.tar.gz \
    && tar xvf libnice-0.1.18.tar.gz \
    && cd libnice-0.1.18 \
    && meson --prefix=/usr --libdir=/usr/lib64 builddir \
    && ninja -C builddir \
    && ninja -C builddir install

#### Boring SSL
RUN cd $BUILD_SRC \
    && git clone https://boringssl.googlesource.com/boringssl \
    && cd boringssl/ \
    && sed -i s/" -Werror"//g CMakeLists.txt \
    && mkdir build \
    && cd build/ \
    && cmake -DCMAKE_CXX_FLAGS="-lrt" .. \
    && make \
    && mkdir -p /opt/boringssl/lib \
    && cd .. \
    && cp -R include /opt/boringssl/ \
    && cp build/ssl/libssl.a /opt/boringssl/lib/ \
    && cp build/crypto/libcrypto.a /opt/boringssl/lib/

### srtp
RUN cd $BUILD_SRC \
    && curl -qL https://github.com/cisco/libsrtp/archive/v2.2.0.tar.gz -o libsrtp-2.2.0.tar.gz \
    && tar xfv libsrtp-2.2.0.tar.gz \
    && cd libsrtp-2.2.0 \
    && ./configure --prefix=/usr --enable-openssl --libdir=/usr/lib64 \
    && make shared_library \
    && make install

### Janus
RUN cd $BUILD_SRC \
    && git clone https://github.com/Bnei-Baruch/janus-gateway.git -b bb \
    && cd janus-gateway \
    && ./autogen.sh \
    && ./configure --prefix=/usr/janusgxy --disable-rabbitmq --disable-docs --enable-websockets --enable-boringssl --enable-dtls-settimeout \
    && make \
    && make install \
    && make configs \
    && chown -R janus:janus /usr/janusgxy

USER janus

EXPOSE 7708
EXPOSE 8088
EXPOSE 8188

CMD ["/usr/janusgxy/bin/janus"]
