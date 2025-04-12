FROM scratch
ADD bs.tar /
COPY create-rootfs.sh /create-rootfs.sh
RUN /create-rootfs.sh
ENV ENV=/etc/profile
ENV BASH_ENV=/etc/profile
