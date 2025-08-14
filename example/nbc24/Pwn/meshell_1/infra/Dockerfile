FROM debian:bookworm-slim

RUN apt-get update && apt-get -y install socat 

RUN mkdir /app
COPY ./meshell.sh /app/
COPY ./flag.txt /app/
RUN chown -R root:root /app && chmod 0555 /app/meshell.sh

RUN useradd --no-create-home user
USER user

EXPOSE 1337
ENTRYPOINT /usr/bin/socat TCP-LISTEN:1337,fork,reuseaddr,bind=0.0.0.0 EXEC:'stdbuf -i0 -o0 -e0 app/meshell.sh',stderr,pty
