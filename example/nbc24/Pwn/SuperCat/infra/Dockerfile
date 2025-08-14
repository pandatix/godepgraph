FROM debian:bookworm-slim

# Update apt
RUN apt update

# Install ssh
RUN apt install -y openssh-server

# Remove apt lists to reduce image size
RUN rm -rf /var/lib/apt/lists/*

# Enable password authentication
RUN sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config
RUN sed -i 's/#Port 22/Port 10201/' /etc/ssh/sshd_config

# Patch missing privilege separation
RUN mkdir /var/run/sshd
RUN chmod 0755 /var/run/sshd

# Add a user named "chall"
RUN useradd user -m -s /bin/bash
RUN useradd user-cracked -s /bin/bash
RUN echo 'user:user' | chpasswd

# Copy flag
COPY flag.txt /home/user/flag.txt
RUN chown root:user-cracked /home/user/flag.txt
RUN chmod 740 /home/user/flag.txt

# Copy program and suid
COPY super_cat /home/user/super_cat
COPY super_cat.rs /home/user/super_cat.rs
RUN chown user-cracked:user-cracked /home/user/super_cat &&\
    chown root:root /home/user/super_cat.rs
RUN chmod 744 /home/user/super_cat.rs &&\
    chmod 6755 /home/user/super_cat

USER root
# SSH port
EXPOSE 10201

# Launch ssh server on port 10201
CMD ["/usr/sbin/sshd", "-D"]
