FROM scratch
COPY create-rootfs.sh /create-rootfs.sh
RUN /create-rootfs.sh
