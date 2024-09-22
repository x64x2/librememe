FROM gcr.io/distroless/static-debian12:nonroot
ARG TARGETARCH
ENV SERVE_BIND=0.0.0.0 SERVE_PORT=8080
COPY linux-${TARGETARCH}/myfans /myfans
WORKDIR /
ENTRYPOINT [ "/myfans" ]
EXPOSE 8080
