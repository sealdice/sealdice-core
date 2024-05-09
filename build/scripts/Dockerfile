FROM scratch
ARG TARGETARCH

COPY ./${TARGETARCH}  .
COPY --chmod=0755 ./${TARGETARCH}/sealdice-core  .

EXPOSE 3211

ENTRYPOINT ["./sealdice-core", "--container-mode"]