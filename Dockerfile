FROM scratch
ENTRYPOINT ["/containerd-auto-configurer"]
COPY containerd-auto-configurer /
