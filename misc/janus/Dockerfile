FROM centos:8

LABEL maintainer="Edo Shor <edoshor@gmail.com>"
LABEL description="Provides an image with BB fork of Janus Gateway for Galaxy 3"

# docker build arguments
ARG BUILD_SRC="/usr/local/src"

RUN yum install -y epel* \
    && yum upgrade -y \
    && yum install -y \
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
    curl \
    golang \
    coturn \
    glib2-devel \
    jansson-devel \
    libcurl-devel \
    which \
    && dnf --enablerepo=PowerTools install -y \
    libconfig-devel \
    libmicrohttpd-devel \
    # texinfo is because of https://savannah.gnu.org/bugs/?56931
    texinfo

RUN /usr/sbin/groupadd -r janus \
    && /usr/sbin/useradd -r -g janus janus \
    && mkdir -p $BUILD_SRC

### Jansson
RUN cd $BUILD_SRC \
    && curl -qL http://www.digip.org/jansson/releases/jansson-2.12.tar.gz -o jansson-2.12.tar.gz \
    && tar xfv jansson-2.12.tar.gz \
    && cd jansson-2.12 \
    && ./configure --prefix=/usr --libdir=/usr/lib64 \
    && make \
    && make install

### getopt
RUN cd $BUILD_SRC \
    && curl -qL http://ftp.gnu.org/gnu/gengetopt/gengetopt-2.23.tar.xz -o gengetopt-2.23.tar.xz \
    && tar xfv gengetopt-2.23.tar.xz \
    && cd gengetopt-2.23 \
    && ./configure --prefix=/usr --libdir=/usr/lib64 \
    && make \
    && make install

### srtp
RUN cd $BUILD_SRC \
    && curl -qL https://github.com/cisco/libsrtp/archive/v2.2.0.tar.gz -o libsrtp-2.2.0.tar.gz \
    && tar xfv libsrtp-2.2.0.tar.gz \
    && cd libsrtp-2.2.0 \
    && ./configure --prefix=/usr --enable-openssl --libdir=/usr/lib64 \
    && make shared_library \
    && make install

### usrsctp
RUN cd $BUILD_SRC \
    && git clone https://github.com/sctplab/usrsctp \
    && cd usrsctp \
    && ./bootstrap \
    && ./configure --prefix=/usr --libdir=/usr/lib64 \
    && make \
    && make install

### libwebsockets
RUN cd $BUILD_SRC \
    && git clone git://git.libwebsockets.org/libwebsockets \
    && cd libwebsockets \
    && git checkout v2.4-stable \
    && mkdir build \
    && cd build \
    && cmake -DLWS_MAX_SMP=1 -DCMAKE_INSTALL_PREFIX:PATH=/usr -DLIB_SUFFIX=64 -DCMAKE_C_FLAGS="-fpic" .. \
    && make \
    && make install

### nice
RUN cd $BUILD_SRC \
    && curl -qL https://nice.freedesktop.org/releases/libnice-0.1.16.tar.gz -o libnice-0.1.16.tar.gz \
    && tar xvf libnice-0.1.16.tar.gz \
    && cd libnice-0.1.16 \
    && ./configure --prefix=/usr --libdir=/usr/lib64 \
    && make \
    && make install

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

### Janus
RUN cd $BUILD_SRC \
    && git clone https://github.com/Bnei-Baruch/janus-gateway.git -b bb \
    && cd janus-gateway \
    && ./autogen.sh \
    && ./configure --prefix=/usr/janusgxy --disable-rabbitmq --disable-mqtt --disable-docs --enable-websockets --enable-boringssl --enable-dtls-settimeout \
    && make \
    && make install \
    && make configs \
    && chown -R janus:janus /usr/janusgxy

USER janus

EXPOSE 7708
EXPOSE 8088
EXPOSE 8188

CMD ["/usr/janusgxy/bin/janus"]
