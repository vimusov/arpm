FROM scratch
ENV ENV=/etc/profile
ENV BASH_ENV=/etc/profile
COPY create-rootfs.sh /create-rootfs.sh
RUN /create-rootfs.sh
