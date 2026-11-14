ARG TOOLCHAIN
FROM ${TOOLCHAIN} AS test
WORKDIR /src
COPY . .
ARG QUALITY_BASE_REF
ARG RELEASE_COMMIT
RUN git config --global --add safe.directory /src \
    && base="$QUALITY_BASE_REF" \
    && if ! git cat-file -e "${base}^{commit}" 2>/dev/null; then base=$(git rev-list --max-parents=0 HEAD | head -n1); fi \
    && QUALITY_BASE_REF="$base" make quality \
    && RELEASE_COMMIT="$RELEASE_COMMIT" bash scripts/package-firmware.sh \
    && cd build/release && sha256sum --check SHA256SUMS \
    && test "$(cat COMMIT)" = "$RELEASE_COMMIT"

FROM scratch
COPY --from=test /src/build/release/ /
