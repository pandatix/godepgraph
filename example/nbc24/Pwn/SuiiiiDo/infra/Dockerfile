FROM debian:bookworm-slim

# Update apt
RUN apt update

# Install packages
RUN apt install -y sudo less
COPY sudoers /etc/sudoers

# Install ssh
RUN apt install -y openssh-server

# Remove apt lists to reduce image size
RUN rm -rf /var/lib/apt/lists/*

# Enable password authentication
RUN sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config
RUN sed -i 's/#Port 22/Port 10200/' /etc/ssh/sshd_config

# Patch missing privilege separation
RUN mkdir /var/run/sshd
RUN chmod 0755 /var/run/sshd

# Add a user named "chall"
RUN useradd user -m -s /bin/bash
RUN useradd user-cracked -m -s /bin/bash
RUN echo 'user:user' | chpasswd

# Flag and notes
COPY flag.txt /home/user-cracked/beca179417c329c2-flag-a2344ce653189f9f.txt
COPY note.txt /home/user-cracked/note.txt
COPY note-user.txt /home/user/note.txt

RUN chown root:user-cracked -R /home/user-cracked/ &&\
    chown root:user -R /home/user/

RUN chmod 750 -R /home/user-cracked/ &&\
    chmod 750 -R /home/user/

# Disable access to /tmp and /var/tmp for chall user
RUN chmod 700 /tmp /var/tmp && \
    chown root:root /tmp /var/tmp

USER root
# SSH port
EXPOSE 10200

# Launch ssh server on port 10200
CMD ["/usr/sbin/sshd", "-D"]
