FROM debian:bookworm-slim

# Update apt
RUN apt update

# Install socat
RUN apt install -y socat

# Remove apt lists to reduce image size
RUN rm -rf /var/lib/apt/lists/*

# Copy liberte to /app
COPY heaper_fort /app/

# Copy flag to /flag.txt
COPY flag.txt /
RUN chmod 744 /flag.txt

# Add a user named "chall"
RUN useradd chall

# Make /app/liberte executable by chall
RUN chown root:root /app/heaper_fort && chmod 755 /app/heaper_fort

# Disable access to /tmp and /var/tmp for chall user
RUN chmod 700 /tmp /var/tmp && \
    chown root:root /tmp /var/tmp


# Switch to chall user
USER chall

# Launch socat on port 10203
CMD ["socat", "TCP-LISTEN:10203,reuseaddr,fork", "EXEC:/app/heaper_fort,stderr"]
