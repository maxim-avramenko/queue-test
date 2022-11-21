FROM php:7.1-fpm

ARG TIME_ZONE=Europe/Moscow
ENV TZ=${TIME_ZONE}

RUN ln -snf /usr/share/zoneinfo/${TZ} /etc/localtime && echo ${TZ} > /etc/timezone \
    && apt-get update \
    && mkdir -p /usr/share/man/man1 \
    && mkdir -p /usr/share/man/man7 \
    && mkdir -p /usr/share/man/man8 \
    && apt-get install -y --no-install-recommends \
        curl \
        debhelper \
        g++ \
        git \
        libc6 \
        libmemcached-dev \
        libcurl4-openssl-dev \
        libfreetype6-dev \
        libjpeg62-turbo-dev \
        libmcrypt-dev \
        libpng-dev \
        libxml2-dev \
        libicu-dev \
        libzip-dev \
        libpq-dev \
        nano \
        procps \
        tzdata \
        unzip \
        wget \
        zlib1g-dev \
    && docker-php-ext-install -j$(nproc) \
        curl \
        opcache \
        pdo \
        pdo_mysql \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* /usr/share/doc/* \
    && curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer

RUN pecl install xdebug-2.9.8 \
    && pecl install xhprof-2.3.5 \
    && pecl install memcached \
    && docker-php-ext-enable \
    xdebug \
    xhprof \
    memcached

#&& docker-php-ext-enable xdebug

COPY ./opcache.ini ${PHP_INI_DIR}/conf.d/opcache.ini
COPY ./xdebug.ini ${PHP_INI_DIR}/conf.d/xdebug.ini
COPY ./php.ini ${PHP_INI_DIR}/php.ini


CMD ["php-fpm"]


