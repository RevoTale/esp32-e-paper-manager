# Official TinyGo toolchain; verified Debian Trixie base in the source environment.
FROM ghcr.io/tinygo-org/tinygo:0.42.0
USER root
ENV GOTOOLCHAIN=go1.26.8
RUN go version
RUN apt-get update && apt-get install --yes --no-install-recommends \
    build-essential bison ca-certificates ccache clang-tidy-19 cmake flex git gperf \
    libffi-dev libssl-dev libusb-1.0-0 ninja-build python3 python3-pip python3-venv wget \
    && rm -rf /var/lib/apt/lists/*
ARG ESP_IDF_VERSION=v5.5.5
ARG ESP_IDF_COMMIT=b774170ff46c393eeb5e495ea37936038d3f4f4f
ENV IDF_PATH=/opt/esp-idf
ENV IDF_TOOLS_PATH=/opt/espressif
# Python is used only by official ESP-IDF build tools, not application runtime.
RUN git clone --branch "${ESP_IDF_VERSION}" --depth 1 \
    https://github.com/espressif/esp-idf.git "${IDF_PATH}" \
    && test "$(git -C "${IDF_PATH}" rev-parse HEAD)" = "${ESP_IDF_COMMIT}" \
    && git -C "${IDF_PATH}" submodule update --init --recursive --depth 1 \
    && "${IDF_PATH}/install.sh" esp32
RUN /bin/bash -c '. "$IDF_PATH/export.sh" && idf.py --version'
